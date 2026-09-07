package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/drkgrntt/htmx-test/database"
	"github.com/drkgrntt/htmx-test/models"
	"github.com/drkgrntt/htmx-test/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/mailgun/mailgun-go/v4"
)

// MailController implements a reply-relay for @<domain> addresses:
//
//   - A stranger emails an alias (e.g. hey@derekgarnett.com). A Mailgun route
//     forwards it here; we stash who it's from/which alias it hit in the
//     email_relays table and forward the content to the personal inbox, with
//     Reply-To rewritten to relay+<row id>@<domain>.
//   - Replying from the personal inbox lands on that relay+<id>@ address.
//     Another Mailgun route forwards *that* here too; we look up the row and
//     send a fresh message out to the original external address, From the
//     original alias — so the personal address never appears anywhere.
//
// Both routes point at the same webhook URL; see the setup notes in
// CLAUDE.md for the exact Mailgun route configuration this depends on.
type MailController struct {
}

const relayLocalPartPrefix = "relay+"

func NewMailController(views fiber.Router, api fiber.Router) *MailController {
	mc := &MailController{}
	mc.registerApiRoutes(api)

	return mc
}

func (c *MailController) registerApiRoutes(api fiber.Router) {
	m := api.Group("/mail")
	m.Post("/webhook", c.handleWebhook)
}

// webhookPayload covers the fields Mailgun's route "forward()" action posts
// (as multipart/form-data or, in some setups, url-encoded — BodyParser
// handles either via the same "form" tags).
type webhookPayload struct {
	Recipient      string `form:"recipient"`
	Sender         string `form:"sender"`
	From           string `form:"from"`
	Subject        string `form:"subject"`
	BodyPlain      string `form:"body-plain"`
	StrippedText   string `form:"stripped-text"`
	MessageHeaders string `form:"message-headers"`
	Timestamp      string `form:"timestamp"`
	Token          string `form:"token"`
	Signature      string `form:"signature"`
}

func (c *MailController) handleWebhook(ctx *fiber.Ctx) error {
	config := utils.GetConfig()
	mg := mailgun.NewMailgun(config.MgDomain, config.MgApiKey)

	payload := new(webhookPayload)
	if err := ctx.BodyParser(payload); err != nil {
		log.Println("mail webhook: failed to parse payload:", err)
		return ctx.SendStatus(http.StatusBadRequest)
	}

	if !verifySignature(config.MgWebhookSigningKey, payload.Timestamp, payload.Token, payload.Signature) {
		log.Println("mail webhook: rejected unverified request")
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	localPart, _, _ := strings.Cut(payload.Recipient, "@")
	if id, ok := strings.CutPrefix(localPart, relayLocalPartPrefix); ok {
		return c.handleRelayReply(ctx, mg, payload, id)
	}

	return c.handleFreshInbound(ctx, mg, payload)
}

// handleFreshInbound handles a stranger's first (or Nth, un-replied-to)
// message to one of our aliases: record it and forward it to the personal
// inbox with a Reply-To that routes any reply back through us.
func (c *MailController) handleFreshInbound(ctx *fiber.Ctx, mg mailgun.Mailgun, payload *webhookPayload) error {
	config := utils.GetConfig()

	externalAddress, externalName := parseMailAddress(payload.From)
	if externalAddress == "" {
		externalAddress = payload.Sender
	}

	relay := models.EmailRelay{
		AliasAddress:    payload.Recipient,
		ExternalAddress: externalAddress,
		ExternalName:    externalName,
		Subject:         payload.Subject,
		MessageId:       extractHeader(payload.MessageHeaders, "Message-Id"),
	}

	db := database.GetDatabase()
	err := db.Get(&relay, `
		INSERT INTO email_relays (alias_address, external_address, external_name, subject, message_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING *`,
		relay.AliasAddress, relay.ExternalAddress, relay.ExternalName, relay.Subject, relay.MessageId,
	)
	if err != nil {
		log.Println("mail webhook: failed to store relay row:", err)
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	replyTo := relayLocalPartPrefix + relay.Id.String() + "@" + config.MgDomain

	displayFrom := externalAddress + " via " + relay.AliasAddress
	if externalName != "" {
		displayFrom = externalName + " (" + externalAddress + ") via " + relay.AliasAddress
	}

	body := payload.StrippedText
	if body == "" {
		body = payload.BodyPlain
	}

	// From must stay on our own verified domain (it's what recipients'
	// mail servers check against SPF/DKIM) — the original sender's name
	// and address are carried in the display name and body instead.
	message := mg.NewMessage(displayFrom+" <"+relay.AliasAddress+">", payload.Subject, body, config.RecipientEmail)
	message.SetReplyTo(replyTo)

	return c.send(ctx, mg, message)
}

// handleRelayReply handles a reply from the personal inbox to a
// relay+<id>@<domain> address: look up who it should really go to, and send
// it out from the original alias.
func (c *MailController) handleRelayReply(ctx *fiber.Ctx, mg mailgun.Mailgun, payload *webhookPayload, id string) error {
	config := utils.GetConfig()

	// Only the personal inbox may drive an outbound send through a relay
	// address. Without this check, anyone who obtained (or guessed) a
	// relay+<id>@ address could send mail "from" our domain as us.
	fromAddress, _ := parseMailAddress(payload.From)
	if !strings.EqualFold(fromAddress, config.RecipientEmail) {
		log.Println("mail webhook: rejected relay reply from unexpected sender:", fromAddress)
		return ctx.SendStatus(http.StatusForbidden)
	}

	db := database.GetDatabase()
	var relay models.EmailRelay
	if err := db.Get(&relay, "SELECT * FROM email_relays WHERE id = $1", id); err != nil {
		log.Println("mail webhook: unknown relay id:", id, err)
		return ctx.SendStatus(http.StatusNotFound)
	}

	subject := payload.Subject
	if subject == "" {
		subject = relay.Subject
	}

	body := payload.StrippedText
	if body == "" {
		body = payload.BodyPlain
	}

	message := mg.NewMessage(relay.AliasAddress, subject, body, relay.ExternalAddress)
	if relay.MessageId != "" {
		message.AddHeader("In-Reply-To", relay.MessageId)
		message.AddHeader("References", relay.MessageId)
	}

	if err := c.send(ctx, mg, message); err != nil {
		return err
	}

	db.MustExec("UPDATE email_relays SET last_used_at = NOW() WHERE id = $1", relay.Id)
	return nil
}

func (c *MailController) send(ctx *fiber.Ctx, mg mailgun.Mailgun, message *mailgun.Message) error {
	mgCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, _, err := mg.Send(mgCtx, message); err != nil {
		log.Println("mail webhook: send failed:", err)
		return ctx.SendStatus(http.StatusInternalServerError)
	}

	return ctx.SendStatus(http.StatusOK)
}

// verifySignature checks a webhook request's signature per Mailgun's scheme:
// HMAC-SHA256(timestamp+token) keyed by the account's HTTP webhook signing
// key (Account Settings → API Security — a distinct secret from the API key
// used for sending; mailgun-go's own VerifyWebhookSignature helper signs
// with the API key instead, which never matches).
func verifySignature(signingKey, timestamp, token, signature string) bool {
	if signingKey == "" || timestamp == "" || token == "" || signature == "" {
		return false
	}

	h := hmac.New(sha256.New, []byte(signingKey))
	h.Write([]byte(timestamp))
	h.Write([]byte(token))
	expected := h.Sum(nil)

	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(expected, got) == 1
}

// parseMailAddress pulls the bare address and display name out of a header
// value like `"Jane Doe" <jane@example.com>`, falling back to treating the
// whole trimmed value as the address if it doesn't parse.
func parseMailAddress(raw string) (address string, name string) {
	parsed, err := mail.ParseAddress(raw)
	if err != nil {
		return strings.TrimSpace(raw), ""
	}
	return parsed.Address, parsed.Name
}

// extractHeader pulls one header value out of Mailgun's "message-headers"
// field, a JSON array of [name, value] pairs.
func extractHeader(headersJSON, key string) string {
	var pairs [][]string
	if err := json.Unmarshal([]byte(headersJSON), &pairs); err != nil {
		return ""
	}
	for _, pair := range pairs {
		if len(pair) == 2 && strings.EqualFold(pair[0], key) {
			return pair[1]
		}
	}
	return ""
}
