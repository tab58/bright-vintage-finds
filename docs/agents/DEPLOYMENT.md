# How to Deploy this Application

Production runtime, configuration, and deploy procedure for `bright-vintage-finds`.
Written 2026-09-15. Values here were read from the live systems on that date —
treat anything marked *(unverified)* as a lead, not a fact.

`AGENTS.md` remains the source of truth for repo layout and code behavior. This
file covers only what lives outside the repo: accounts, DNS, env vars, and the
sequence for getting a change into production.

---

## 1. What runs where

Two deployable services, both on Railway, both fronted by Cloudflare.

| Service | Railway service | Image | Serves |
|---|---|---|---|
| API | `bright-vintage-finds-api` | `ghcr.io/tab58/main-api` | `/healthz`, `/admin/*` (inventory API) |
| Frontend | `bright-vintage-finds-frontend` | `ghcr.io/tab58/public-site` | splash at `/`, inventory PWA at `/inventory*`, proxies `/admin/*` |

Plus `db-main-api-production` (Postgres) and `Cloudflared` (tunnel) in the same
Railway project.

### Request paths

The inventory PWA and the admin API share one origin. This is deliberate: it
makes the Cloudflare Access cookie first-party and removes CORS entirely.

```
phone/browser
   |
   |-- https://brightvintagefinds.com/           -> Caddy -> static splash        (public)
   |-- https://brightvintagefinds.com/inventory  -> Caddy -> PWA shell            (Access)
   |-- https://brightvintagefinds.com/admin/*    -> Caddy -> reverse_proxy -> API (Access)
   |
   `-- https://api.brightvintagefinds.com/admin/*  -> CF Tunnel -> API            (Access)
```

The `api.` hostname still works for direct API calls (curl, debugging). The
browser app does not use it.

---

## 2. Accounts and identifiers

- **Railway** — account `timabright@gmail.com`, project `bright-vintage-finds`
  (`58f17c21-8f91-4bac-8b2f-28042151592c`), environment `production`
  (`6eda5a44-b6cb-4bca-bfd0-6af9cda950b8`).
  Verify with `railway whoami` before running anything.
- **Cloudflare** — account `timabright@gmail.com`
  (`b1c12628e7697d1178530ed619485f87`), zone `brightvintagefinds.com`,
  Zero Trust team `cool-scene-e6df`.
- **GHCR** — images under `ghcr.io/tab58/`, pushed by CI with `GITHUB_TOKEN`.

Railway service IDs (needed by the deploy workflow):

| Service | ID |
|---|---|
| `bright-vintage-finds-api` | `b4a1b93c-5a38-47d3-9c21-ad6cb3baef43` |
| `bright-vintage-finds-frontend` | `e57b76dd-9df8-48c7-9571-95909b7f6dbf` |

---

## 3. DNS and ingress

Five records on the `brightvintagefinds.com` zone:

| Name | Type | Content | Proxy |
|---|---|---|---|
| `brightvintagefinds.com` | CNAME | `gg11n5o0.up.railway.app` | Proxied |
| `api.brightvintagefinds.com` | Tunnel | `brightvintagefinds-main` | Proxied |
| `*.brightvintagefinds.com` | A | `91.195.240.94` | Proxied |
| `www.brightvintagefinds.com` | A | `91.195.240.94` | Proxied |
| `_railway-verify.brightvintagefinds.com` | TXT | `railway-verify=...` | DNS only |

Two different ingress mechanisms, which is easy to get wrong:

- **The frontend uses a Railway public domain.** The apex is a CNAME to
  `gg11n5o0.up.railway.app`, with `_railway-verify` proving ownership. It does
  *not* go through the tunnel.
- **The API is tunnel-only.** `api.` is a Cloudflare Tunnel record pointing at
  the `brightvintagefinds-main` tunnel, which routes to the API's Railway
  private domain `bright-vintage-finds.railway.internal:8080`. The API has no
  public Railway domain.

`www` redirects to the apex via the zone's single active redirect rule,
"Redirect from WWW to root [Template]": matches `URI Full wildcard
r"https://www.*"` and issues a 301 to `wildcard_replace(http.request.full_uri,
...)`. Zone → Rules → Overview (filtered to Redirect Rules).

---

## 4. Configuration

### 4.1 Railway — `bright-vintage-finds-frontend`

| Variable | Value | Why |
|---|---|---|
| `API_UPSTREAM` | `http://bright-vintage-finds.railway.internal:8080` | Caddy `reverse_proxy` target for `/admin/*`. Private networking is IPv6-only. |
| `BACKEND_API` | *(empty)* | Injected into `/env.js`. Empty means the client uses a same-origin base. Set it only to point the app at a different origin. |
| `PORT` | set by Railway | Caddy listen port. |

The private domain is `bright-vintage-finds.railway.internal`, **not** the
service name `bright-vintage-finds-api` — the service was renamed and the
private domain kept the old name. Using the service name yields a 502.

### 4.2 Railway — `bright-vintage-finds-api`

| Variable | Value |
|---|---|
| `ENV` | `production` |
| `SERVER_PORT` | `8080` |
| `MAIN_DB_URL` | Postgres connection string *(secret)* |
| `AWS_REGION` | `auto` |
| `S3_BASE_ENDPOINT` | `https://t3.storageapi.dev` |
| `S3_UPLOAD_BUCKET` | `stocked-cube-vvxsqadtfydz` |
| `CF_ACCESS_TEAM_DOMAIN` | `cool-scene-e6df.cloudflareaccess.com` |
| `CF_ACCESS_AUD` | `9f0a4a04b0bc6cbe6c7735d639b3a1e671c38180d99dc40a533ffeac833ff4e4` |
| AWS credentials | standard `AWS_*` vars *(secret)* |

The service dumps its non-secret config at boot, so `railway logs` shows the
effective values. Three of these fail the boot outright if wrong — see §8.

### 4.3 GitHub — `production` environment

Variables: `RAILWAY_MAIN_API_SERVICE_ID`, `RAILWAY_MAIN_API_ENVIRONMENT_ID`,
`RAILWAY_PUBLIC_SITE_SERVICE_ID`, `RAILWAY_PUBLIC_SITE_ENVIRONMENT_ID`.

Secrets: `RAILWAY_API_TOKEN` (project-scoped; sent as `Project-Access-Token`),
`MAIN_DB_URL` (used by the migration workflow only).

---

## 5. Cloudflare Access

One self-hosted app, `main-api-admin`
(`78ed1f9c-01d5-4626-8fcd-adeb810111c8`), with three destinations:

- `api.brightvintagefinds.com/admin`
- `brightvintagefinds.com/admin`
- `brightvintagefinds.com/inventory`

All three share one policy, `allow-owner`, and therefore one AUD:
`9f0a4a04b0bc6cbe6c7735d639b3a1e671c38180d99dc40a533ffeac833ff4e4`.

**Keep them on one app.** Adding destinations to the existing app leaves the AUD
unchanged, so `CF_ACCESS_AUD` needs no edit and the API needs no redeploy. A
separate app means a new AUD, and `internal/cfaccess` is fail-closed in
production — a mismatch 401s every admin request.

Defense in depth: Cloudflare verifies the policy at the edge and injects
`Cf-Access-Jwt-Assertion`; `internal/cfaccess` independently verifies that JWT
against the team JWKS and the AUD. The backend never trusts the edge blindly.

The splash page at `/` is deliberately outside Access and stays public.

**To see or change who can log in:** Zero Trust dashboard → Access controls →
Policies → `allow-owner`. The include rules there are the allow-list. Changes
take effect immediately; no deploy needed. Access controls → Applications →
`main-api-admin` shows which policies are attached, and Insights & Logs → Logs
→ Access shows actual login attempts.

---

## 6. Deploying

### 6.1 The normal path: push to `main`

`.github/workflows/deploy.yml` runs on every push to `main`. `dorny/paths-filter`
decides what rebuilds:

| Changed path | Triggers |
|---|---|
| `backend/main-api/**` | unit tests -> release -> image -> deploy API |
| `frontend/public_site/**` | release -> image -> deploy frontend |
| anything else (CI, docs, AGENTS.md) | nothing builds |

Per service the pipeline is: semantic-release (tags `main-api/v*` or
`public-site/v*`) -> Docker build+push to GHCR -> `_deploy-railway.yml`.
**Deploy only fires when semantic-release publishes a new version**, which
requires a conventional-commit subject (`feat:`, `fix:`, ...). A `chore:` or
`docs:` commit builds nothing.

So: commit with a `feat:`/`fix:` prefix, push to `main`, watch the run.

```bash
gh run watch "$(gh run list --workflow=deploy.yml --limit 1 \
  --json databaseId --jq '.[0].databaseId')" --exit-status
```

### 6.2 Manual deploy and rollback

`_deploy-railway.yml` is `workflow_dispatch`-able. Use it to redeploy without a
new release, or to roll back to any tag already in GHCR:

```bash
gh workflow run _deploy-railway.yml \
  -f service=public-site \
  -f version=1.1.2
```

`service` is `main-api` or `public-site`; `version` is a bare semver that exists
as a GHCR tag. This is the recovery path when a release published but the deploy
failed — re-running `deploy.yml` will not help, because semantic-release finds
no new commits and skips everything downstream.

### 6.3 The rollout check

`.github/scripts/railway-deploy.sh` points the service at the image, triggers a
redeploy, then **polls the deployment to a terminal status**. `SUCCESS` passes;
`FAILED`/`CRASHED`/`REMOVED`/`SKIPPED` fails the job; otherwise it polls to
`DEPLOY_TIMEOUT_SECONDS` (default 600, `POLL_INTERVAL_SECONDS` default 10).

This matters because Railway keeps the previous container serving when a new one
crash-loops. Without the wait, a broken release reports green while the old
image quietly keeps running. If a deploy fails:

```bash
railway logs <deployment-id> --service bright-vintage-finds-api -d --lines 30
```

The failure message prints the deployment ID. Check the status classification
offline with `bash .github/scripts/railway-deploy.sh --self-test`.

### 6.4 Database migrations

`main-api_migrate_db.yml` is **manual and separate from deploys**, and runs
`atlas migrate apply` against the production DB using the `MAIN_DB_URL` secret.

```bash
gh workflow run main-api_migrate_db.yml
```

**Run the migration before deploying code that needs the new schema.** The API
seeds builtin selling places at boot and fails fast on error, so shipping a
release ahead of its migration crash-loops the container.

---

## 7. Verifying a deploy

```bash
# frontend up, splash public
curl -s -o /dev/null -w '%{http_code}\n' https://brightvintagefinds.com/

# runtime config: expect BACKEND_API = "" and cache-control: no-store
curl -s -D- https://brightvintagefinds.com/env.js

# admin behind Access: expect 302 to cool-scene-e6df.cloudflareaccess.com
curl -s -o /dev/null -w '%{http_code} %{redirect_url}\n' \
  https://brightvintagefinds.com/admin/items

# API itself
curl -s -o /dev/null -w '%{http_code}\n' https://api.brightvintagefinds.com/healthz

# container actually booted
railway logs --service bright-vintage-finds-api -d --lines 20
```

On a phone: open `https://brightvintagefinds.com/inventory`, complete the Access
login, then Share -> Add to Home Screen. The manifest installs as "Inventory"
with `start_url` `/inventory`.

---

## 8. Failure modes seen in production

Each of these actually happened; the symptom is what you will see first.

| Symptom | Cause | Fix |
|---|---|---|
| `/admin/*` returns `404 page not found` (plain text, from Go) | API booted healthz-only: `MAIN_DB_URL` empty, so admin routes never registered | Set `MAIN_DB_URL`, redeploy |
| Deploy green but behavior unchanged; container uptime far older than the deploy | New container crash-looped, Railway kept the old one | Now caught by the rollout check (§6.3) — read the deployment logs |
| `relation "selling_places" does not exist` | Code deployed ahead of its migration | Run the migration (§6.4), redeploy |
| `HeadBucket ... 404, NotFound` on a bucket name starting with `$` | Malformed `S3_UPLOAD_BUCKET`; S3 names cannot contain `$` | Correct the variable, redeploy |
| `/admin/*` returns 401 `missing or invalid Cloudflare Access assertion` | Request reached the API without an Access assertion, or `CF_ACCESS_AUD` does not match the app | Confirm the hostname is a destination on `main-api-admin` and the AUDs match (§5) |
| Stale `BACKEND_API` served for hours after a deploy | Cloudflare cached `/env.js` by its `.js` extension | Fixed by `Cache-Control: no-store` in the Caddyfile; purge the URL once to evict an existing entry |
| Caddy `502` to the API | Used the service name instead of the private domain | Use `bright-vintage-finds.railway.internal` (§4.1) |

---

## 9. Access limitations for agents

Claude Code's auto mode denies some Railway calls:

- `railway variables --service bright-vintage-finds-api` — *Credential Materialization*
- `railway variables set ...` (any service) — *Modify Shared Resources*

Reading the frontend's variables and `railway logs` do work. Env-var changes
have to be made by the repo owner, in the dashboard or CLI. Cloudflare has no
CLI path here at all — `cloudflared` does not manage Access apps, and no API
token is configured — so Access changes are dashboard-only.
