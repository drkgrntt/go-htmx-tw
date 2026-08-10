package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/drkgrntt/htmx-test/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/mailgun/mailgun-go/v4"
)

type UtilsController struct {
}

func NewUtilsController(views fiber.Router, api fiber.Router) *UtilsController {
	ac := &UtilsController{}
	ac.registerApiRoutes(api)

	return ac
}

func (c *UtilsController) registerApiRoutes(api fiber.Router) {
	contact := api.Group("/utils")
	contact.Post("/log", c.emailLog)
}

type EmailLogBody struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (c *UtilsController) emailLog(ctx *fiber.Ctx) error {
	body := new(EmailLogBody)
	err := ctx.BodyParser(body)
	if err != nil {
		log.Fatal(err)
	}

	config := utils.GetConfig()

	mg := mailgun.NewMailgun(config.MgDomain, config.MgApiKey)
	sender := "Email Logging <logs@derekgarnett.com>"
	subject := body.Subject
	emailBody := body.Body
	recipient := config.RecipientEmail

	// The message object allows you to add attachments and Bcc recipients
	message := mg.NewMessage(sender, subject, emailBody, recipient)

	mgCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10 second timeout
	resp, id, err := mg.Send(mgCtx, message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)

	return ctx.SendStatus(http.StatusNoContent)
}
