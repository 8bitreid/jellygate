# jellygate

A simple, secure invite management system for [Jellyfin](https://jellyfin.org). Admins generate unique invite links; anyone with a link can self-register a Jellyfin account.

## features

- one-time or multi-use invite links with optional expiry
- library access scoping per invite
- admin dashboard authenticated via your existing Jellyfin admin credentials
- discord webhook notifications on invite creation and registration
- sits behind traefik with automatic TLS

## stack

- go 1.25 — stdlib `net/http`, `html/template`, `embed`
- postgresql — schema managed with `golang-migrate`
- docker compose

## configuration

Copy `.env.example` to `.env` and fill in:

```env
JELLYFIN_URL=http://localhost:8096
LISTEN_ADDR=:8080
SECRET_KEY=<64 random hex chars — run: openssl rand -hex 32>
DATABASE_URL=postgres://user:pass@db:5432/jellygate?sslmode=disable
DISCORD_WEBHOOK_URL=        # optional
```

## running

```bash
docker compose up -d
```

Access the admin dashboard at your configured domain (or `http://localhost:8080` locally). Log in with your Jellyfin admin credentials.

## development

Requires Go 1.25+ and Docker (for integration tests).

```bash
# run all tests
go test ./...

# build binary
go build -o jellygate ./cmd/server
```

## project layout

```
internal/domain/        — types and port interfaces (no external deps)
internal/store/         — postgresql implementations
internal/jellyfin/      — jellyfin api client
internal/notifications/ — discord webhook notifier
internal/auth/          — session and csrf management
internal/middleware/    — http middleware (auth, rate limiting, security headers)
internal/handler/       — http handlers
web/                    — html templates and css
migrations/             — embedded sql migration files
```

## stages

| # | stage | status |
|---|-------|--------|
| 1 | skeleton & domain contracts | ✅ |
| 2 | postgresql store | ✅ |
| 3 | jellyfin api client | 🔄 |
| 4 | auth & middleware | ⬜ |
| 5 | admin handlers + ui | ✅ |
| 6 | invite registration flow | ⬜ |
| 7 | discord notifications | ⬜ |
| 8 | dockerfile + compose + traefik | ⬜ |
