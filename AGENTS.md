# AGENTS.md

This repo documents itself through `AGENTS.md` files. They are the source of truth for how the codebase is laid out and how each part behaves. Treat them as authoritative over assumptions or stale memory.

## Project Overview

bright-vintage-finds is an early-stage monorepo for a vintage-goods selling platform: a public-facing website combined with an inventory system for the site. The owner uploads pictures and details of items to sell, gains insight into their own sales, and may eventually get a sales portal. Currently one Go service exists: `backend/main-api`, an HTTP API built on the external `github.com/tab58/huma-http-server` framework (huma-based server with JWT auth and router plumbing). It boots a server, registers a `/healthz` route, loads config from env vars via Viper, and guards `/admin` routes by verifying Cloudflare Access assertions (`internal/cfaccess`; open in development, fail-closed in production when `CF_ACCESS_TEAM_DOMAIN`/`CF_ACCESS_AUD` are unset). It also serves an unauthenticated read-only catalog under `/public/*` for the shop front page. `frontend/` holds one web client, `frontend/public_site/` (Vite + React + TypeScript + Tailwind CSS v4 + shadcn/ui): the public shop front page at `/` (hero, filterable catalog grid with cursor paging, photo lightbox, Whatnot call-to-action) plus the owner-facing inventory PWA at `/inventory*` (routing, typed API client, and the three inventory pages: intake, status-grouped search list, item detail) talking to the `/admin` inventory API. Items flow draft → listed ("Active" in the UI) → sold, with archived as a side exit and restore back to draft; the client owns those rules, the API only validates the status value. The draft → listed move goes through a dialog that captures the listing price (required), since a listed item is immediately on the shop front. `environment/local-docker/` holds the local Docker environment (Postgres + Atlas migration runner + floci S3 + main-api), started with `task up` from the repo root.

## Reference Documentation

Do **not** read these eagerly. Read them on demand when the task calls for the information they cover.

- **[backend/main-api/Taskfile.yml](backend/main-api/Taskfile.yml)** — Run/test/codegen/migration tasks for main-api.
- **[backend/main-api/db/README.md](backend/main-api/db/README.md)** — Persistence layer (Ent + Atlas): folder structure, mixins, ER diagram, schema-change workflow.
- **[docs/agents/FUNCTIONAL_DESIGN.md](docs/agents/FUNCTIONAL_DESIGN.md)** — What the product does and the rules it enforces: actors, surfaces, item lifecycle, owner and shopper workflows, photo handling, API contract, degraded modes, deliberate behaviours and known gaps.
- **[backend/main-api/testing/REGISTRY.md](backend/main-api/testing/REGISTRY.md)** — main-api business rule registry: every rule the product enforces (backend and PWA), where it is enforced, the test that proves it, and the gaps. Add or update the rules a feature touches before implementing it, and update their status before opening a PR.

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
│       ├── api/                 # HTTP driving adapter: NewServer (framework wiring) only
│       │   └── routes/          # Every huma route: handlers, wire DTOs, mappers, domain→HTTP error mapping
│       ├── db/                  # Persistence layer: Ent schemas, generated client, Atlas migrations (see db/README.md)
│       ├── internal/app/        # Application core (ports and adapters) — see internal/app/AGENTS.md
│       ├── internal/cfaccess/   # Cloudflare Access guard: verifies Cf-Access-Jwt-Assertion on /admin routes
│       ├── internal/logger/     # slog-based JSON logger
│       ├── Taskfile.yml         # run / test / generate / migration tasks
│       └── main-api.Dockerfile  # Multi-stage build → scratch image (build context = repo root)
├── frontend/                    # Web clients, one folder per client
│   └── public_site/             # Shop front page (/) + inventory PWA (/inventory*), Vite + React + TS + Tailwind v4 + shadcn/ui
│       ├── src/App.tsx          # Router: shop front at /, inventory pages under /inventory
│       ├── src/pages/           # Home (public shop front), Inventory (status-grouped list, long-press moves), Intake (photos + all fields → draft), Item (read-only detail + Edit-details toggle; places/labels/Whatnot/notes always editable; status actions, list-it and mark-sold dialogs)
│       ├── src/components/      # Shared UI: item-fields.tsx (the item form, used by intake + item detail), list-it-dialog.tsx (listing-price prompt, used by Inventory + Item), chip-picker.tsx
│       ├── src/components/shop/ # Shop front pieces: catalog.tsx (filters + grid + paging), item-card.tsx, item-lightbox.tsx, config.ts (hardcoded copy + Whatnot handle), format.ts
│       ├── src/components/ui/   # shadcn/ui primitives (generated by the shadcn CLI; re-add, don't hand-roll)
│       ├── src/index.css        # Tailwind import + shadcn design tokens (CSS variables)
│       ├── components.json      # shadcn CLI config (new-york style, neutral base, @/* → src/*)
│       ├── .prettierrc          # Prettier config (semicolons, single quotes); .prettierignore skips dist/ and src/components/ui/
│       ├── src/api/client.ts    # Typed fetch client for the /admin inventory API (shared request/unwrap helpers; re-runs Access login on an expired session)
│       ├── src/api/public.ts    # Typed fetch client for the /public catalog API
│       ├── src/assets/          # Site art (texture-band.png in the shop footer; card-bright.png is unused since the splash page was replaced)
│       ├── public/env.js        # Dev default for window.BACKEND_API (prod: Caddy injects it)
│       ├── Caddyfile            # Serves dist/ on $PORT; reverse-proxies /admin/* and /public/* to $API_UPSTREAM; /env.js exposes $BACKEND_API at runtime
│       ├── public_site.Dockerfile # bun build → caddy:2-alpine (build context = this dir)
│       └── vite.config.ts       # Tailwind v4 plugin + PWA (install-only; /inventory* navigations bypass the cached shell) + '@' → /src alias + dev proxy /admin and /public → localhost:3000
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
- **api/** — the HTTP **driving adapter**, split in two. `api/register.go` is only `NewServer`: it wraps `huma-http-server`'s `server.New`, puts `/healthz` on the auth/logging skip list, and calls the `Register*` functions in `api/routes`. Everything else lives in **api/routes/**, which owns the wire format and nothing else: request/response DTOs, `huma.Register` calls, mappers to and from the application's types, and the `domain.Error` → HTTP status mapping in `errors.go`. `RegisterHealthz` serves the platform healthcheck (apps must not register their own). Given an `*app.Application` the rest register the inventory admin routes (Cloudflare-Access-guarded `/admin/*`): items CRUD + search filters + status moves + delete-if-never-listed and mark-sold (`items.go`; `PATCH` carries `status`, stamping/clearing `listed_at`, and can correct `sold_*` fields after a sale), multipart image upload (`item_images.go`, registered only when `app.HasImageStore()`), and selling-places and labels CRUD (`selling_places.go`, `labels.go`), which are thin pass-throughs on the application because they have no rules beyond storage. Alongside those is the unauthenticated public catalog (`public_items.go`): `GET /public/items` (listed items only, cursor-paginated by KSUID id with `query`/`category`/`label` filters), `GET /public/filters` (the categories and labels present among listed items), and `GET /public/items/{id}/images` (photo URLs, 404 unless the item is listed). These use their own output structs — a public response never carries acquisition cost, notes, Whatnot numbers, selling places, sold data, or S3 keys. Every rule the handlers used to hold now lives in `internal/app`. Huma convention: response structs must have a `Body` field, otherwise the framework emits 204 with struct fields as headers.
- **internal/app/** — the application core, in ports-and-adapters form: `domain/` (types, parsers, status and deletion rules; no Ent, no huma), `ports/` (repository and storage interfaces, plus in-memory fakes), `adapters/ent` and `adapters/s3` (driven adapters), and package `app` itself (`app.go`, `items.go`, `catalog.go`, `images.go`, `views.go`) — one root object, `Application`, carrying every use case the HTTP adapter calls. It imports only `domain` and `ports`; `cmd/app/main.go` is the composition root that builds the adapters and passes them to `app.New`. See `internal/app/AGENTS.md`.
- **db/** — Package `db_platform`: Ent schemas (`schema/` + `schema/mixin/`), generated client (`generated/`, never hand-edit), Atlas migrations (`migrations/`), pgx-backed client wrapper (`client.go`; `Raw()` exposes the pool for test harnesses). Entities: User, Item, ItemImage, SellingPlace, Label. Reached only through the repository ports in `internal/app/ports`, implemented by `internal/app/adapters/ent`; wired when `MAIN_DB_URL` is set (required in production). See `db/README.md`.
- **internal/logger/** — slog JSON logger with configurable level and extra handlers.
- **Object storage** — uses the shared `aws_s3` client from `environment/shared/golang/clients/aws_s3` (Railway bucket in prod, floci locally; credentials via standard AWS env vars). When `S3_BASE_ENDPOINT` is set, boot builds the client and `Ping`s (HeadBucket) the upload bucket so misconfiguration fails fast; when unset, storage is skipped and the server still boots. Image uploads raise `humago.MultipartMaxMemory` to the 15 MB upload cap so multipart bodies are buffered in memory: the production image is `FROM scratch` and has no `/tmp` for Go's multipart spill files.

The HTTP framework (huma server, router, JWT auth middleware, `AuthInfoBuilder`) lives in the external module `github.com/tab58/huma-http-server`, not in this repo.

## Key Technologies

**Backend:** Go 1.27, huma v2 (via `tab58/huma-http-server`), Viper (config), slog (logging), JWT auth (golang-jwt via framework), Ent ORM + Atlas migrations (Postgres, pgx driver, KSUID ids), aws-sdk-go-v2 (S3 object storage). Planned per Taskfile/config: Redis/Asynq, AWS Secrets Manager, Firebase.

**Frontend:** Vite 5, React 18, TypeScript, Tailwind CSS v4 (via `@tailwindcss/vite`), shadcn/ui components (`bunx --bun shadcn@latest add <name>`), lucide-react icons. Package manager is Bun (`bun.lock`; there is no `package-lock.json`). Formatting is Prettier (`bun run format` / `bun run format:check`); the shadcn-generated `src/components/ui/` is ignored so re-adding a component does not churn. Commands (from `frontend/public_site/`): `bun install` / `bun run dev` / `bun run build` / `bun run preview`.

## CI/CD

GitHub Actions (`.github/workflows/`), modeled on stack-prime, production-only (no beta images, no staging):

- **unit-tests.yml** — PRs to main touching `backend/main-api/**`: runs Go unit tests via reusable `_go-unit-tests.yml`.
- **deploy.yml** — push to main: `dorny/paths-filter` detects which service changed, then per service: semantic-release (`_go-release-docker.yml` — release + Docker steps only, nothing Go-specific despite the name) → image to GHCR → deploy to Railway production (`_deploy-railway.yml` + `scripts/railway-deploy.sh`). Deploy only fires when a new release is published. `railway-deploy.sh` polls the deployment to a terminal status and fails the job on `FAILED`/`CRASHED` (`DEPLOY_TIMEOUT_SECONDS`, default 600) — a crash-looping container leaves the previous one serving, which otherwise reports as a green deploy. `bash .github/scripts/railway-deploy.sh --self-test` checks the status classification offline.
  - **main-api** (`backend/main-api/**`): unit tests first, tag `main-api/v<version>`, image `ghcr.io/tab58/main-api`, `.releaserc.json` in the service dir.
  - **public-site** (`frontend/public_site/**`): no test suite, tag `public-site/v<version>`, image `ghcr.io/tab58/public-site` (`public_site.Dockerfile`: `oven/bun:1-alpine` runs `bun install --frozen-lockfile` + `bun run build` → Caddy serving `dist/`; `Caddyfile` reads `PORT`, reverse-proxies `/admin/*` and `/public/*` to `API_UPSTREAM` (the API's Railway private domain), and serves `/env.js` with the `BACKEND_API` env var injected at runtime).
- **main-api_migrate_db.yml** — manual (workflow_dispatch) Atlas migration apply against production DB (`MAIN_DB_URL` secret).
- **ghcr-cleanup.yml** — nightly GHCR retention (currently `dry-run: true`).

Full production configuration (accounts, service IDs, DNS records, per-service env vars, Cloudflare Access) and the deploy/rollback/migration procedure live in `docs/agents/DEPLOYMENT.md`.

Required GitHub config: `production` environment with vars `RAILWAY_MAIN_API_SERVICE_ID`, `RAILWAY_MAIN_API_ENVIRONMENT_ID`, `RAILWAY_PUBLIC_SITE_SERVICE_ID`, `RAILWAY_PUBLIC_SITE_ENVIRONMENT_ID` and secrets `RAILWAY_API_TOKEN`, `MAIN_DB_URL`.

Production ingress uses two mechanisms: `brightvintagefinds.com` is a proxied CNAME to the frontend's Railway public domain (`gg11n5o0.up.railway.app`), while `api.brightvintagefinds.com` is a Cloudflare Tunnel record (`brightvintagefinds-main`, cloudflared service in the same Railway project) routing to the API's private domain `bright-vintage-finds.railway.internal:8080` — the API has no public Railway domain. The inventory PWA and the admin API share the `brightvintagefinds.com` origin: Caddy reverse-proxies `/admin/*` to the API, so the Cloudflare Access cookie is first-party and no CORS preflight is involved. The `main-api-admin` Access app protects three destinations — `api.brightvintagefinds.com/admin`, `brightvintagefinds.com/admin`, and `brightvintagefinds.com/inventory` — under one policy (`allow-owner`) and therefore one AUD, which the API's `CF_ACCESS_TEAM_DOMAIN`/`CF_ACCESS_AUD` match so the in-app cfaccess guard verifies the same tokens. The shop front page at `/` and the `/public/*` catalog API it reads stay public.

An Access session eventually expires, and Cloudflare answers the next request with a cross-origin 302 to its login page — which a `fetch` cannot follow, so the PWA handles that itself. `src/api/client.ts` sends `redirect: 'manual'` and treats an `opaqueredirect` response as "signed out": it reloads the page once per tab session (a `sessionStorage` flag, cleared by the first response that reaches the origin, keeps a failed round-trip from looping). The reload only works because `/inventory*` is in the service worker's `navigateFallbackDenylist`: otherwise the cached shell answers the navigation, the browser never reaches Cloudflare, and the app comes up signed out with every `/admin` call dying as `TypeError: Failed to fetch`. How long the session lasts is the Access app's session duration, set in the Zero Trust dashboard, not in this repo.

## Code Generation

- `task generate` runs `go generate ./...` — currently only Ent codegen (`db/generate.go` → `db/generated/`).
- Schema-change workflow (codegen → migration diff → apply) is documented in `db/README.md`.

## Prerequisites

- Go 1.27+
- Task runner: `brew install go-task`
- dotenvx (env var loading for `task run`)
- Atlas CLI: `brew install ariga/tap/atlas`
- Docker (integration tests, Atlas dev DB)
- Bun (frontend package manager and script runner): `brew install oven-sh/bun/bun`

## Known Drift / TODOs

- `.env.development` is required by `task run` but is not checked in.
- `MAIN_DB_URL` is required in production; optional in development (boots healthz-only without it).
- The inventory PWA lives inside `public_site/` (route `/inventory*`); it deploys with the existing public-site pipeline. A browser/phone pass-through of the intake flow is still pending.
- The shop front page's "where to buy" copy and Whatnot handle are hardcoded in `frontend/public_site/src/components/shop/config.ts`, with the handle left as a placeholder. `selling_places` rows carry only a name, so venue details cannot come from the database yet.
- The public catalog serves 15-minute presigned S3 URLs, so photos are re-fetched from storage on every page view and cannot be CDN-cached or bookmarked. Proxying or a public prefix is the fix when traffic makes it matter.
