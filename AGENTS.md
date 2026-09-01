# AGENTS.md

This repo documents itself through `AGENTS.md` files. They are the source of truth for how the codebase is laid out and how each part behaves. Treat them as authoritative over assumptions or stale memory.

## Project Overview

bright-vintage-finds is an early-stage monorepo for a vintage-goods selling platform: a public-facing website combined with an inventory system for the site. The owner uploads pictures and details of items to sell, gains insight into their own sales, and may eventually get a sales portal. Currently one Go service exists: `backend/main-api`, an HTTP API built on the external `github.com/tab58/huma-http-server` framework (huma-based server with JWT auth and router plumbing). It boots a server, registers a `/healthz` route, loads config from env vars via Viper, and guards `/admin` routes by verifying Cloudflare Access assertions (`internal/cfaccess`; open in development, fail-closed in production when `CF_ACCESS_TEAM_DOMAIN`/`CF_ACCESS_AUD` are unset). `frontend/` is an empty placeholder for the web client to come. `environment/local-docker/` holds the local Docker environment (Postgres + Atlas migration runner + floci S3 + main-api), started with `task up` from the repo root.

## Reference Documentation

Do **not** read these eagerly. Read them on demand when the task calls for the information they cover.

- **[backend/main-api/Taskfile.yml](backend/main-api/Taskfile.yml)** — Run/test/codegen/migration tasks for main-api.
- **[backend/main-api/db/README.md](backend/main-api/db/README.md)** — Persistence layer (Ent + Atlas): folder structure, mixins, ER diagram, schema-change workflow.
- **[backend/main-api/test-cases.md](backend/main-api/test-cases.md)** — main-api test-case log: per-feature record of what is tested at each pyramid level. Add an entry before implementing a feature.

## Repository Structure

```
/
├── backend/
│   └── main-api/                # Go HTTP API (sole service so far)
│       ├── cmd/app/             # Entry point (main.go) + config/ (Viper env loading)
│       ├── api/                 # Server construction + /healthz registration
│       ├── db/                  # Persistence layer: Ent schemas, generated client, Atlas migrations (see db/README.md)
│       ├── internal/cfaccess/   # Cloudflare Access guard: verifies Cf-Access-Jwt-Assertion on /admin routes
│       ├── internal/logger/     # slog-based JSON logger
│       ├── Taskfile.yml         # run / test / generate / migration tasks
│       └── main-api.Dockerfile  # Multi-stage build → scratch image (build context = repo root)
├── frontend/                    # Empty placeholder (web client TBD)
└── environment/
    └── local-docker/            # Local Docker Compose: db-main-api (Postgres 17), db-main-api-migrate (Atlas), floci (S3), main-api, cloudflared (opt-in tunnel, profile "tunnel")
```

## Common Commands

### Local environment (repo root)
```bash
task up    # Start local Docker stack (dotenvx loads environment/local-docker/.env.development)
task down  # Stop it
task up -- cloudflared  # Additionally start the Cloudflare tunnel (needs CLOUDFLARED_TUNNEL_TOKEN in .env.development)
```

### main-api (backend/main-api/)
```bash
task run                    # Run API server (dotenvx with .env.development)
task run-tests              # Unit + integration tests
task run-unit-tests         # Unit tests with coverage
task run-integration-tests  # Integration tests, -p 1 (requires Docker; shared Postgres)
task generate               # go generate (mocks etc.)

# Database migrations (Atlas + Ent, run from backend/main-api/)
task generate-migration -- <name>  # Diff Ent schema → new migration
task apply-migrations              # Apply migrations to local Postgres
task apply-schema-direct           # Apply schema directly, no migrations
```

## Architecture

### main-api Structure
- **cmd/app/** — `main.go`: loads config, constructs server, starts listening. No graceful shutdown yet (marked in code).
- **cmd/app/config/** — Viper-based config. Env vars bound by reflection over `mapstructure` tags; secrets carry `json:"-"` so the startup config dump never logs them. Validates `ENV` (development|production), `SERVER_PORT`, `JWT_SIGNING_SECRET` as required. Declares (not yet used) config for AWS (region, Secrets Manager Firebase key, S3), Redis (Asynq/cache), and `MAIN_DB_URL` (Postgres).
- **api/** — `NewServer` wraps `huma-http-server`'s `server.New`, always skips auth/logging for `/healthz` and registers the platform healthcheck. Apps must not register their own `/healthz`.
- **db/** — Package `db_platform`: Ent schemas (`schema/` + `schema/mixin/`), generated client (`generated/`, never hand-edit), Atlas migrations (`migrations/`), pgx-backed client wrapper (`client.go`). Entities: User, Item, ItemImage. Not yet wired into the server. See `db/README.md`.
- **internal/logger/** — slog JSON logger with configurable level and extra handlers.

The HTTP framework (huma server, router, JWT auth middleware, `AuthInfoBuilder`) lives in the external module `github.com/tab58/huma-http-server`, not in this repo.

## Key Technologies

**Backend:** Go 1.25, huma v2 (via `tab58/huma-http-server`), Viper (config), slog (logging), JWT auth (golang-jwt via framework), Ent ORM + Atlas migrations (Postgres, pgx driver, KSUID ids). Planned per Taskfile/config: Redis/Asynq, AWS (S3, Secrets Manager), Firebase.

**Frontend:** TBD (`frontend/` is empty).

## Code Generation

- `task generate` runs `go generate ./...` — currently only Ent codegen (`db/generate.go` → `db/generated/`).
- Schema-change workflow (codegen → migration diff → apply) is documented in `db/README.md`.

## Prerequisites

- Go 1.25+
- Task runner: `brew install go-task`
- dotenvx (env var loading for `task run`)
- Atlas CLI: `brew install ariga/tap/atlas`
- Docker (integration tests, Atlas dev DB)

## Known Drift / TODOs

- `.env.development` is required by `task run` but is not checked in.
- `db/` exists but is not wired into the server yet (`MAIN_DB_URL` config declared, unused).
- `frontend/` is empty.
