# AGENTS.md

This repo documents itself through `AGENTS.md` files. They are the source of truth for how the codebase is laid out and how each part behaves. Treat them as authoritative over assumptions or stale memory.

## Project Overview

bright-vintage-finds is an early-stage monorepo for a vintage-goods selling platform: a public-facing website combined with an inventory system for the site. The owner uploads pictures and details of items to sell, gains insight into their own sales, and may eventually get a sales portal. Currently one Go service exists: `backend/main-api`, an HTTP API built on the external `github.com/tab58/huma-http-server` framework (huma-based server with JWT auth and router plumbing). It boots a server, registers a `/healthz` route, loads config from env vars via Viper, and guards `/admin` routes by verifying Cloudflare Access assertions (`internal/cfaccess`; open in development, fail-closed in production when `CF_ACCESS_TEAM_DOMAIN`/`CF_ACCESS_AUD` are unset). `frontend/` holds one web client, `frontend/public_site/` (Vite + React + TypeScript + StyleX): the splash page at `/` plus the owner-facing inventory PWA at `/inventory*` (routing, intake/search/mark-sold pages, typed API client) talking to the `/admin` inventory API. `environment/local-docker/` holds the local Docker environment (Postgres + Atlas migration runner + floci S3 + main-api), started with `task up` from the repo root.

## Reference Documentation

Do **not** read these eagerly. Read them on demand when the task calls for the information they cover.

- **[backend/main-api/Taskfile.yml](backend/main-api/Taskfile.yml)** — Run/test/codegen/migration tasks for main-api.
- **[backend/main-api/db/README.md](backend/main-api/db/README.md)** — Persistence layer (Ent + Atlas): folder structure, mixins, ER diagram, schema-change workflow.
- **[backend/main-api/test-cases.md](backend/main-api/test-cases.md)** — main-api test-case log: per-feature record of what is tested at each pyramid level. Add an entry before implementing a feature.

## Repository Structure

```
/
├── .github/
│   ├── workflows/               # CI/CD: unit tests (PR), deploy (push to main), DB migrate, GHCR cleanup
│   ├── scripts/railway-deploy.sh # Points a Railway service at a new image + triggers redeploy
│   └── CODEOWNERS
├── backend/
│   └── main-api/                # Go HTTP API (sole service so far)
│       ├── cmd/app/             # Entry point (main.go) + config/ (Viper env loading)
│       ├── api/                 # Server construction + /healthz registration
│       ├── db/                  # Persistence layer: Ent schemas, generated client, Atlas migrations (see db/README.md)
│       ├── internal/cfaccess/   # Cloudflare Access guard: verifies Cf-Access-Jwt-Assertion on /admin routes
│       ├── internal/logger/     # slog-based JSON logger
│       ├── Taskfile.yml         # run / test / generate / migration tasks
│       └── main-api.Dockerfile  # Multi-stage build → scratch image (build context = repo root)
├── frontend/                    # Web clients, one folder per client
│   └── public_site/             # Splash page (/) + inventory PWA (/inventory*), Vite + React + TS + StyleX
│       ├── src/App.tsx          # Router: splash at /, inventory pages under /inventory
│       ├── src/pages/           # Inventory (search/filter list), Intake (photos + all fields), Item (detail, edit, mark-sold)
│       ├── src/api/client.ts    # Typed fetch client for the /admin inventory API
│       ├── src/assets/          # Splash page art
│       ├── public/env.js        # Dev default for window.BACKEND_API (prod: Caddy injects it)
│       ├── Caddyfile            # Serves dist/ on $PORT; reverse-proxies /admin/* to $API_UPSTREAM; /env.js exposes $BACKEND_API at runtime
│       ├── public_site.Dockerfile # npm build → caddy:2-alpine (build context = this dir)
│       └── vite.config.ts       # StyleX + PWA (install-only) + dev proxy /admin → localhost:3000
└── environment/
    ├── local-docker/            # Local Docker Compose: db-main-api (Postgres 17), db-main-api-migrate (Atlas), floci (S3), main-api
    └── shared/golang/           # Shared Go module (clients/aws_s3: S3 client + mocks), consumed by services via replace directive
```

## Common Commands

### Local environment (repo root)
```bash
task up        # Start local Docker stack (dotenvx loads environment/local-docker/.env.development)
task down      # Stop it
task front-up  # Run the frontend dev server (Vite; prints the local URL)
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
- **cmd/app/config/** — Viper-based config. Env vars bound by reflection over `mapstructure` tags; secrets carry `json:"-"` so the startup config dump never logs them. Validates `ENV` (development|production) and `SERVER_PORT` as required; `MAIN_DB_URL` required in production. Declares config for AWS (region, Secrets Manager Firebase key), S3 (`S3_BASE_ENDPOINT`/`S3_UPLOAD_BUCKET`), Redis (Asynq/cache), and `MAIN_DB_URL` (Postgres); Redis and Secrets Manager are not yet used.
- **api/** — `NewServer` wraps `huma-http-server`'s `server.New`, always skips auth/logging for `/healthz` and registers the platform healthcheck. Apps must not register their own `/healthz`. With an `AppDeps` carrying a DB client it also registers the inventory admin routes (Cloudflare-Access-guarded `/admin/*`): items CRUD + search filters (`items.go`), mark-sold, multipart image upload to S3 (`item_images.go`, registered only when storage is configured), selling-places and labels CRUD (`selling_places.go`, `labels.go`), builtin place seed at boot (`seed.go`), and a lazily-created well-known owner row (`owner.go`). Huma convention: response structs must have a `Body` field, otherwise the framework emits 204 with struct fields as headers.
- **db/** — Package `db_platform`: Ent schemas (`schema/` + `schema/mixin/`), generated client (`generated/`, never hand-edit), Atlas migrations (`migrations/`), pgx-backed client wrapper (`client.go`; `Raw()` exposes the pool for test harnesses). Entities: User, Item, ItemImage, SellingPlace, Label. Wired into the server via `api.AppDeps` when `MAIN_DB_URL` is set (required in production). See `db/README.md`.
- **internal/logger/** — slog JSON logger with configurable level and extra handlers.
- **Object storage** — uses the shared `aws_s3` client from `environment/shared/golang/clients/aws_s3` (Railway bucket in prod, floci locally; credentials via standard AWS env vars). When `S3_BASE_ENDPOINT` is set, boot builds the client and `Ping`s (HeadBucket) the upload bucket so misconfiguration fails fast; when unset, storage is skipped and the server still boots.

The HTTP framework (huma server, router, JWT auth middleware, `AuthInfoBuilder`) lives in the external module `github.com/tab58/huma-http-server`, not in this repo.

## Key Technologies

**Backend:** Go 1.27, huma v2 (via `tab58/huma-http-server`), Viper (config), slog (logging), JWT auth (golang-jwt via framework), Ent ORM + Atlas migrations (Postgres, pgx driver, KSUID ids), aws-sdk-go-v2 (S3 object storage). Planned per Taskfile/config: Redis/Asynq, AWS Secrets Manager, Firebase.

**Frontend:** Vite 5, React 18, TypeScript, StyleX (via `vite-plugin-stylex`). Commands (from `frontend/public_site/`): `npm run dev` / `npm run build` / `npm run preview`.

## CI/CD

GitHub Actions (`.github/workflows/`), modeled on stack-prime, production-only (no beta images, no staging):

- **unit-tests.yml** — PRs to main touching `backend/main-api/**`: runs Go unit tests via reusable `_go-unit-tests.yml`.
- **deploy.yml** — push to main: `dorny/paths-filter` detects which service changed, then per service: semantic-release (`_go-release-docker.yml` — release + Docker steps only, nothing Go-specific despite the name) → image to GHCR → deploy to Railway production (`_deploy-railway.yml` + `scripts/railway-deploy.sh`). Deploy only fires when a new release is published. `railway-deploy.sh` polls the deployment to a terminal status and fails the job on `FAILED`/`CRASHED` (`DEPLOY_TIMEOUT_SECONDS`, default 600) — a crash-looping container leaves the previous one serving, which otherwise reports as a green deploy. `bash .github/scripts/railway-deploy.sh --self-test` checks the status classification offline.
  - **main-api** (`backend/main-api/**`): unit tests first, tag `main-api/v<version>`, image `ghcr.io/tab58/main-api`, `.releaserc.json` in the service dir.
  - **public-site** (`frontend/public_site/**`): no test suite, tag `public-site/v<version>`, image `ghcr.io/tab58/public-site` (`public_site.Dockerfile`: npm build → Caddy serving `dist/`; `Caddyfile` reads `PORT`, reverse-proxies `/admin/*` to `API_UPSTREAM` (the API's Railway private domain), and serves `/env.js` with the `BACKEND_API` env var injected at runtime).
- **main-api_migrate_db.yml** — manual (workflow_dispatch) Atlas migration apply against production DB (`MAIN_DB_URL` secret).
- **ghcr-cleanup.yml** — nightly GHCR retention (currently `dry-run: true`).

Required GitHub config: `production` environment with vars `RAILWAY_MAIN_API_SERVICE_ID`, `RAILWAY_MAIN_API_ENVIRONMENT_ID`, `RAILWAY_PUBLIC_SITE_SERVICE_ID`, `RAILWAY_PUBLIC_SITE_ENVIRONMENT_ID` and secrets `RAILWAY_API_TOKEN`, `MAIN_DB_URL`.

Production ingress: no public Railway domain — a Cloudflare Tunnel (cloudflared service in the same Railway project) routes both hostnames to Railway private domains on port 8080. The inventory PWA and the admin API share the `brightvintagefinds.com` origin: Caddy reverse-proxies `/admin/*` to the API, so the Cloudflare Access cookie is first-party and no CORS preflight is involved. The `main-api-admin` Access app protects three destinations — `api.brightvintagefinds.com/admin`, `brightvintagefinds.com/admin`, and `brightvintagefinds.com/inventory` — under one policy (`allow-owner`) and therefore one AUD, which the API's `CF_ACCESS_TEAM_DOMAIN`/`CF_ACCESS_AUD` match so the in-app cfaccess guard verifies the same tokens. The splash page at `/` stays public.

## Code Generation

- `task generate` runs `go generate ./...` — currently only Ent codegen (`db/generate.go` → `db/generated/`).
- Schema-change workflow (codegen → migration diff → apply) is documented in `db/README.md`.

## Prerequisites

- Go 1.27+
- Task runner: `brew install go-task`
- dotenvx (env var loading for `task run`)
- Atlas CLI: `brew install ariga/tap/atlas`
- Docker (integration tests, Atlas dev DB)

## Known Drift / TODOs

- `.env.development` is required by `task run` but is not checked in.
- `MAIN_DB_URL` is required in production; optional in development (boots healthz-only without it).
- The inventory PWA lives inside `public_site/` (route `/inventory*`); it deploys with the existing public-site pipeline. A browser/phone pass-through of the intake flow is still pending.
