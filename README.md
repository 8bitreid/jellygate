# jellygate

A simple, secure invite management system for [Jellyfin](https://jellyfin.org). Admins generate unique invite links; anyone with a link can self-register a Jellyfin account.

## features

- one-time or multi-use invite links with optional expiry
- library access scoping per invite
- admin dashboard authenticated via your existing Jellyfin admin credentials
- discord webhook notifications on invite creation and user registration
- seerr-style first-run setup — no credentials in env, admin logs in via UI

## stack

- go 1.25 — stdlib `net/http`, `html/template`, `embed`
- postgresql — schema managed with `golang-migrate`
- docker compose

## configuration

Copy `.env.example` to `.env` and fill in:

```env
JELLYFIN_URL=http://localhost:8096
LISTEN_ADDR=:8080
BASE_URL=http://localhost:8080
DATABASE_URL=postgres://jellygate:pass@127.0.0.1:5432/jellygate?sslmode=disable
DB_PASSWORD=changeme
DISCORD_WEBHOOK_URL=        # optional
SECURE_COOKIES=false        # set true when serving over HTTPS
BEHIND_PROXY=false          # set true when behind a reverse proxy
```

## running

```bash
docker compose up -d
```

Access the admin dashboard at `http://localhost:8080`. On first visit you'll be prompted to sign in with your Jellyfin admin credentials to complete setup.

## development

```bash
docker compose -f compose.dev.yaml up --build
```

This builds from source and shares the host network so Jellyfin on `localhost:8097` is reachable.

Run tests directly:

```bash
go test ./...
```

### linting & formatting

Install dependencies:

```bash
npm install
```

Lint HTML templates:

```bash
npm run lint:html
```

Lint JavaScript:

```bash
npm run lint:js
```

Lint everything:

```bash
npm run lint
```

Format HTML templates:

```bash
npm run format:html
```

Format JavaScript:

```bash
npm run format:js
```

Format everything:

```bash
npm run format
```

**Git hooks:** Husky automatically formats and lints staged files on commit.

## project layout

```
cmd/server/             — main entrypoint
internal/domain/        — types and port interfaces (no external deps)
internal/store/         — postgresql implementations + migrations
internal/jellyfin/      — jellyfin api client
internal/notifications/ — discord webhook notifier
internal/auth/          — session and csrf management
internal/middleware/    — http middleware (auth, rate limiting, security headers)
internal/handler/       — http handlers
web/                    — html templates and css
```

## status

### stages

| # | stage | status |
|---|-------|--------|
| 1 | skeleton & domain contracts | ✅ merged |
| 2 | postgresql store | ✅ merged |
| 3 | jellyfin api client | ✅ merged |
| 4 | auth & middleware | ✅ merged |
| 5 | admin handlers + ui | ✅ merged |
| 6 | invite registration flow | ✅ merged |
| 7 | discord notifications | ✅ merged |
| 8 | dockerfile + compose + dev workflow | ✅ merged |

### known issues

| area | issue |
|------|-------|
| notifications | `InviteCreated` notification never fires — admin handler does not receive the notifier |
| dashboard | `ListLibraries` error silently swallowed — empty library list with no user feedback |
| invite handler | `SetLibraryAccess`, `registrations.Create`, and `IncrementUse` failures are not logged |
| invite handler | `mustGenerateToken` panics instead of returning a 500 |
