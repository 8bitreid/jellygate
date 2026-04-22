# jellygate

A self-hosted invite management system for [Jellyfin](https://jellyfin.org). Admins generate shareable invite links; anyone with a link can self-register a Jellyfin account without needing admin access.

## features

- one-time or multi-use invite links with optional expiry
- library access scoping per invite
- admin dashboard authenticated via your existing Jellyfin admin credentials
- discord webhook notifications on invite creation and user registration
- first-run setup wizard — configure Jellyfin URL, Seerr, and Discord from the UI; no service URLs in env

## prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose v2
- A running [Jellyfin](https://jellyfin.org) instance reachable from the jellygate container
- (development only) Go 1.25+

## configuration

Copy `.env.example` to `.env` and fill in:

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | yes | PostgreSQL DSN, e.g. `postgres://jellygate:pass@127.0.0.1:5432/jellygate?sslmode=disable` |
| `DB_PASSWORD` | yes | PostgreSQL password (used by the `db` service) |
| `LISTEN_ADDR` | no | Bind address (default `:8080`) |
| `BASE_URL` | no | Public-facing URL, e.g. `https://invite.example.com` — derived from request host when unset |
| `SECURE_COOKIES` | no | Set `true` when serving over HTTPS (default `true`) |
| `BEHIND_PROXY` | no | Set `true` when behind a reverse proxy for correct IP detection |

Jellyfin URL, Seerr URL, and Discord webhook URL are configured through the UI after first run — not via env vars.

## running

```bash
docker compose up -d
```

The image is published to [GitHub Container Registry](https://github.com/8bitreid/jellygate/pkgs/container/jellygate) on every tagged release. `compose.yaml` pulls `ghcr.io/8bitreid/jellygate:latest` automatically.

On first visit, navigate to `http://localhost:8080` — you'll be redirected to `/setup` to provide your Jellyfin URL and sign in with your Jellyfin admin credentials. Seerr and Discord webhook URLs are optional and can be configured later under **Settings**.

## releases

Releases are automated via GitHub Actions. Push a semver tag to trigger a multi-platform build (`linux/amd64` + `linux/arm64`) and publish to GHCR:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

To redeploy on your server after a new release:

```bash
docker compose pull jellygate
docker compose up -d jellygate
```

## building from source

```bash
go build -o jellygate ./cmd/server
```

To embed a version string (shown in startup logs):

```bash
go build \
  -ldflags="-X main.version=v0.1.0" \
  -o jellygate ./cmd/server
```

The `Dockerfile` passes `VERSION` via a build arg:

```bash
docker build --build-arg VERSION=v0.1.0 -t jellygate .
```

## development

Start a local stack (builds from source, host network so Jellyfin on `localhost:8097` is reachable):

```bash
docker compose -f compose.dev.yaml up --build
```

Run tests (requires Docker — integration tests spin up a real PostgreSQL container via testcontainers):

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

## architecture

jellygate follows a hexagonal (ports & adapters) layout. Domain interfaces live in `internal/domain/` with zero external dependencies; all concrete implementations depend inward.

```
cmd/server/             — wires dependencies, reads env, starts net/http server
internal/domain/        — aggregate types (Invite, Session) and port interfaces
internal/store/         — PostgreSQL implementations; migrations run automatically on startup
internal/jellyfin/      — Jellyfin HTTP client
internal/notifications/ — Discord webhook notifier + no-op implementation
internal/auth/          — session management and CSRF helpers
internal/middleware/    — auth enforcement, setup redirect, rate limiting, security headers
internal/handler/       — HTTP handlers (admin, invite registration, setup, settings, tutorial, health)
web/                    — HTML templates and static assets, embedded into the binary at compile time
```

Database migrations live in `internal/store/migrations/` and are applied automatically via `golang-migrate` on every startup.

## known issues

| area | issue |
|------|-------|
| notifications | `InviteCreated` notification never fires — admin handler does not receive the notifier |
| dashboard | `ListLibraries` error silently swallowed — empty library list with no user feedback |
| invite handler | `SetLibraryAccess`, `registrations.Create`, and `IncrementUse` failures are not logged |
| invite handler | `mustGenerateToken` panics instead of returning a 500 |

## contributing

1. Fork the repo and create a branch from `main`
2. Make your changes — run `go test ./...` and `go vet ./...` before pushing
3. Open a pull request; CI will run tests and linting automatically
