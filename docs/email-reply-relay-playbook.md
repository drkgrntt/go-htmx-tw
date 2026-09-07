# Building an email reply-relay with Mailgun — playbook

How to give a domain masked-alias addresses (`hey@yourdomain.com`) that forward
to a personal inbox, where replying sends back out under the original alias
without ever exposing the personal address. Built for derekgarnett.com
(`controllers/mail.go`, `models/email_relay.go`, `database/migrate.go`); this
doc is the reusable version of what we learned getting it working, so the next
build in another repo skips straight past the mistakes.

## The design (provider-agnostic)

- **Inbound**: stranger emails `hey@yourdomain.com` → your webhook receives
  it → insert a row keyed by its own id (alias, external sender, subject,
  `Message-Id`) → forward the content to the personal inbox, with `Reply-To`
  rewritten to `relay+<row id>@yourdomain.com`.
- **Outbound**: reply from the personal inbox lands on that `relay+<id>@`
  address → your webhook receives *that* too → look up the row → send a
  fresh message out **from the original alias** to the original external
  address, with `In-Reply-To`/`References` set to the stored `Message-Id` for
  threading.
- **One webhook URL handles both directions.** Branch on whether the
  recipient's local part starts with `relay+`.
- **Guard the outbound leg**: only honor a `relay+<id>@` hit whose `From`
  matches the personal inbox address. Without this, anyone who obtains a
  `relay+<id>@` address (e.g. from a leaked email) could send mail "from"
  your domain as you.
- **Verify every request's webhook signature** before trusting or storing
  anything (see the signing-key gotcha below — this is the one most likely
  to bite you again).

## Step-by-step setup (do it in this order)

1. **Pick the domain deliberately.** When you add a domain in Mailgun's
   dashboard, it suggests (and lightly nudges you toward) a subdomain like
   `mg.yourdomain.com`. That's a genuinely separate domain resource from the
   bare apex, not a detail of the same one — MX/SPF/DKIM on `mg.yourdomain.com`
   do nothing for `yourdomain.com` and vice versa. Decide up front:
   - Bare apex (`yourdomain.com`) if you want clean-looking aliases and the
     apex isn't already hosting real mail (check `dig MX yourdomain.com`
     first — if something's already there, adding Mailgun's MX will break it).
   - The `mg.` subdomain if you want sending-reputation isolation from a real
     human mailbox on the apex, and don't mind uglier `hey@mg.yourdomain.com`
     aliases.
   - If you want the apex and Mailgun defaulted you to a subdomain, add the
     apex as its **own additional domain** (dashboard: Sending → Domains →
     Add New Domain, type the bare domain instead of accepting the `mg.`
     suggestion). You can have both at once.
2. **Add the DNS records Mailgun gives you** for that domain: MX (×2,
   `mxa`/`mxb.mailgun.org`, priority 10), SPF TXT, DKIM TXT
   (`<selector>._domainkey`), and the tracking CNAME (optional — only needed
   for click/open tracking, not for anything in this design).
3. **Grab the HTTP webhook signing key now, before writing any code.**
   Dashboard → Account Settings → API Security. It is a **separate secret
   from the sending API key** — Mailgun signs webhook requests with this key,
   not the one you use to call the send API. This is the single most likely
   thing to trip you up (it did here): older client library helpers (at
   least mailgun-go v4.12.0's `VerifyWebhookSignature`/`VerifyWebhookRequest`)
   compute the HMAC using the **API key** instead, which will never validate
   a real request. Don't trust a library helper's webhook-verification
   method without reading what secret it actually hashes against — implement
   it yourself if there's any doubt:
   ```go
   h := hmac.New(sha256.New, []byte(webhookSigningKey)) // NOT the API key
   h.Write([]byte(timestamp))
   h.Write([]byte(token))
   expected := h.Sum(nil)
   got, _ := hex.DecodeString(signature)
   valid := subtle.ConstantTimeCompare(expected, got) == 1
   ```
4. **Write one Mailgun Route** matching every recipient on the domain(s) in
   play, forwarding to your webhook:
   - `match_recipient(".*@yourdomain\.com")` — **this is regex, not a
     glob.** `*@yourdomain.com` (no leading `.`) is invalid/meaningless as a
     regex and will silently never match anything. Always start it with
     `.*`, and escape literal dots (`\.`).
   - If your alias addresses and your `relay+` addresses end up on
     *different* domains (e.g. aliases on the apex, relay+ still on an old
     `mg.` domain during a migration), match both:
     `.*@(yourdomain\.com|mg\.yourdomain\.com)`. Simplest is to keep
     everything on one domain so this never comes up.
   - Action: `forward("https://yourapp.com/api/mail/webhook")`.
   - Note this *is* the webhook — Mailgun's separate "Webhooks" dashboard
     feature (for open/click/bounce event tracking) is unrelated and you
     don't need it for this.
5. **Deploy, then test with a real email**, not just a curl to your own
   endpoint (a curl only proves your endpoint is reachable and rejects
   unsigned requests — it proves nothing about DNS/routing/signing).

## Debugging when a test email doesn't show up

Work through these in order — each rules out a whole layer:

1. **Check your app's logs first** for any hit to the webhook route at all.
   Nothing logged means the message never reached your server — look at
   Mailgun, not your code.
2. **Check for a bounce** in the account you sent the test from. The exact
   SMTP error text is the fastest diagnostic you'll get:
   - **No MX / domain not found** — DNS not propagated yet, or you sent to
     the wrong domain (e.g. the apex when only the `mg.` subdomain has MX).
   - **`550 5.7.1 Relaying denied`** — the message *reached* Mailgun's
     inbound MTA and got rejected there, before your route/webhook was ever
     consulted. This means Mailgun doesn't (yet) recognize that recipient's
     domain as one of your active receiving domains. Causes, roughly in
     order of likelihood:
     - The domain was verified very recently. The dashboard/API can report a
       domain as `active` before that status has propagated out to the
       shared `mxa`/`mxb` MTA fleet that serves every Mailgun customer —
       this has taken up to about an hour in practice. Wait and retry.
     - Your route's expression doesn't actually match the recipient (see the
       regex gotcha above).
3. **Query Mailgun's own event log directly** rather than guessing — it's
   the ground truth for what actually happened to the message:
   ```
   curl -s --user "api:$MG_API_KEY" \
     "https://api.mailgun.net/v3/<domain>/events?limit=25" | python3 -m json.tool
   ```
   - **Zero events at all** → the message never reached Mailgun. DNS/MX
     problem, not a routing or code problem.
   - **A rejected/failed event** → reached Mailgun, got bounced there — read
     the reason in the event.
   - **An accepted/forwarded event, but your webhook still shows nothing or
     rejects it** → the problem is entirely in your endpoint (auth/parsing),
     not in Mailgun's config.
4. **If your webhook is getting hit but rejecting with "unverified"**, add a
   temporary log line dumping the raw payload fields (recipient, subject,
   timestamp, token, signature, `Content-Type`) before the signature check,
   and wait for Mailgun's automatic retry (routes retry a failed forward on
   their own schedule — we saw ~10 minutes apart initially — so you don't
   need to send a fresh test email, the same one comes back). This tells you
   in one shot whether the fields are parsing correctly (form vs multipart —
   in practice we saw `application/x-www-form-urlencoded`, not multipart,
   for the `forward()` action) and rules field-parsing bugs in or out before
   you go chasing the signing-key issue above.
5. You can drive all of the above (domain creation, route inspection/fixes,
   event queries) straight from the `mailgun-go` SDK already in this repo's
   `go.mod` with a disposable `go run` script using the existing
   `utils.LoadConfig` — no need to touch the dashboard by hand or wait on UI
   propagation to check API-visible state.

## Mistakes we actually made, for pattern-matching next time

1. Typed `match_recipient("*@domain.com")` in the dashboard (glob habit) —
   valid-looking, silently never matched.
2. Assumed the domain Mailgun auto-suggested (`mg.derekgarnett.com`) *was*
   the apex domain for receiving purposes — it's a fully separate domain
   resource; mail to the bare apex had nowhere to go until we added it too.
3. Used the Mailgun **API key** for webhook signature verification because
   that's what the SDK helper uses — the account actually signs with a
   separate **HTTP webhook signing key** that was sitting unused in the
   dashboard the whole time.
