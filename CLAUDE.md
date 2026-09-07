# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

The Go backend for derekgarnett.com — a Fiber (`gofiber/fiber/v2`) server rendering `templ` components server-side, sprinkled with HTMX on the frontend and Tailwind (via CDN script, no build step) for styling. It serves a personal landing page, tech/music sub-pages, a contact form (emails via Mailgun), a simple blog with an admin-only create/edit flow, and an email reply-relay (see below) that lets `*@derekgarnett.com` aliases forward to a personal inbox and reply back out without exposing it.

## Commands

Development (hot reload, requires [air](https://github.com/air-verse/air)):
```
air
```
This runs `templ generate && go build -o ./tmp/main .` on every change to `.go`/`.templ` files (see `.air.toml`).

Manual build/run:
```
templ generate    # regenerate *_templ.go from *.templ files — required after editing any .templ file
go build -o ./tmp/main .
./tmp/main
```

There is no test suite, linter config, or CI in this repo.

Docker:
```
docker compose up --build
```
Uses `network_mode: host` so the container's Postgres connection (`DB_HOST=localhost` in `.env`) and nginx reverse proxy keep working unchanged from the pre-Docker systemd setup. `.env` is loaded via `env_file` in `docker-compose.yml`; `godotenv.Load()` in `utils/config.go` tolerates a missing `.env` file (expected inside the container) but fatals on any other load error.

## Configuration

Environment variables (see `.env`, not committed) are loaded once into a package-level `Config` singleton by `utils.LoadConfig` and read anywhere via `utils.GetConfig()`. Key vars: `PORT`, `ENVIRONMENT` (`development` unlocks `/routes` and blog write endpoints), `MG_DOMAIN`/`MG_API_KEY`/`RECIPIENT_EMAIL` (Mailgun), `DB_HOST`/`DB_PORT`/`DB_NAME`/`DB_USER`/`DB_PASSWORD`.

## Architecture

Request flow: `main.go` → `controllers.InitializeControllers` → per-feature `*Controller` → `handlers` (render templ pages) or inline controller logic (form parsing, DB writes, Mailgun sends).

- **`controllers/`** — one file per feature area (`landing`, `blog`, `contact`, `utils`). Each defines a `New*Controller(views, api fiber.Router)` that registers both HTML view routes (under `views`, the root group) and JSON/form API routes (under `api`, mounted at `/api`). `controller.go` wires them all together in `registerRoutes` and also exposes `/metrics` (Fiber monitor) and, non-production only, `/routes` (dumps the full route table as JSON).
- **`handlers/`** — thin adapters that call into `templ` components and wrap them via `adaptor.HTTPHandler(templ.Handler(...))` to satisfy Fiber's handler signature. Controllers call handlers to render full pages; handlers take no persistence responsibility themselves.
- **`components/`** — `templ` templates, one subdirectory per feature (`layout`, `landing`, `tech`, `music`, `blog`, `contact`). Each `.templ` file has a generated `*_templ.go` sibling — **never edit the generated file directly**; edit the `.templ` and run `templ generate`. `layout.Render(c)` is the shared page shell (nav, HTMX/Tailwind/EasyMDE CDN scripts, footer) that every page component wraps itself in.
- **`models/`** — plain structs with `db:"..."` struct tags for `sqlx` scanning, plus view-helper methods (e.g. `Blog.ContentPreview()`, `Blog.IsPublished()`) called directly from templates.
- **`database/`** — `Connect()` opens the `sqlx` Postgres connection into a package-level `*sqlx.DB` (`database.GetDatabase()`) and immediately calls `Migrate()`, which runs an idempotent `CREATE TABLE IF NOT EXISTS` schema string. There is no migration framework — schema changes are made by editing `database/migrate.go`'s `schema` string directly.
- **`utils/`** — config loading, a `LogFatalError` helper (many call sites `log.Fatal` on error rather than returning it — this is existing style, not something to silently "fix" unless asked), and small map helpers.

Admin/write access to the blog (`POST /api/blog`, `PUT /api/blog/:id`, and the view routes that render create/edit forms) is gated by `config.Environment == "development"`, not real auth — there's no user/session system in this app. The nav's "Thoughts" link and admin affordances key off the same check via `c.Locals("IsAdmin")` set in `main.go`.

## Email reply-relay

`controllers/mail.go` (`MailController`) implements a masked-alias relay so `hey@derekgarnett.com` (or any made-up `*@derekgarnett.com` alias) can forward to `RECIPIENT_EMAIL` and be replied to without ever revealing that address to the outside sender. Both directions hit the same public webhook, `POST /api/mail/webhook`:

- **Inbound** (stranger → alias): a Mailgun route forwards the message here. A row is inserted into `email_relays` (alias, external sender, subject, `Message-Id`) and the content is forwarded to `RECIPIENT_EMAIL`, with `Reply-To` rewritten to `relay+<row id>@<MG_DOMAIN>`.
- **Outbound** (reply from the personal inbox): replying lands on that `relay+<id>@` address, which a second Mailgun route also forwards here. The row is looked up by `<id>` and a fresh message is sent out **from the original alias** to the original external address, with `In-Reply-To`/`References` set for threading. Guarded so only mail whose `From` matches `RECIPIENT_EMAIL` can trigger an outbound send — otherwise a leaked `relay+<id>@` address could be used to send mail "from" the domain as us.

Every request is verified against Mailgun's webhook signature (`mailgun.VerifyWebhookSignature`, HMAC over `timestamp`+`token` keyed by `MG_API_KEY`) before anything is trusted or stored.

This depends on one-time Mailgun configuration that lives outside the codebase, not in this repo:
1. **Receiving must be enabled for the domain** (MX records pointed at Mailgun) — separate from whatever's already configured for sending.
2. **One route**, matching every recipient on the domain and forwarding to the webhook:
   - expression: `match_recipient(".*@derekgarnett\.com")`
   - action: `forward("https://derekgarnett.com/api/mail/webhook")`
   - It catches both fresh mail to any alias and replies to `relay+<id>@` addresses — `handleWebhook` tells them apart by the `relay+` recipient prefix, so no second route is needed.

No new env vars — it reuses `MG_DOMAIN`, `MG_API_KEY`, and `RECIPIENT_EMAIL`.
