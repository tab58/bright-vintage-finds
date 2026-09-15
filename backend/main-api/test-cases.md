# main-api test-case log

Per-feature record of what is tested at each pyramid level and why.
Add an entry before implementing a feature; update its status before opening a PR.

## Cloudflare Access admin-route guard (`internal/cfaccess`)

**Status:** implemented

Middleware that verifies the `Cf-Access-Jwt-Assertion` header (RS256 JWT from
Cloudflare Access) on `/admin` and `/admin/*` paths. Unconfigured + development
= open (local dev has no CF edge); unconfigured + production = fail closed.

| Level | Case | Why |
|-------|------|-----|
| unit | non-admin path passes through untouched | public API must not be affected |
| unit | valid assertion on admin path → next called | happy path |
| unit | missing header on admin path → 401 | primary lockout |
| unit | expired token → 401 | replay protection |
| unit | wrong audience → 401 | token minted for another Access app must not work |
| unit | token signed by unknown key → 401 | forged token protection |
| unit | unconfigured + dev bypass → admin path open | local development without CF |
| unit | unconfigured + production → admin path 401 | fail closed at trust boundary |
| unit | half-configured (one of team domain / AUD) → constructor error | misconfig fails at boot, not at request time |
| integration | none | middleware has no DB/S3 dependency; JWKS fetch covered by unit tests via local test server |
| contract/E2E | none yet | no admin routes exist; add an E2E through the CF edge when the first admin route ships |

## Object storage boot wiring (shared `aws_s3` client)

**Status:** implemented

main-api uses the shared S3 client from
`environment/shared/golang/clients/aws_s3` (Railway bucket in production, floci
locally). When `S3_BASE_ENDPOINT` is set, boot builds the client and `Ping`s
(HeadBucket) `S3_UPLOAD_BUCKET`, so bad credentials fail fast instead of at
first upload. Client behavior (endpoint/path-style options, Ping) is unit
tested in the shared module (`clients/aws_s3/client_test.go`).

| Level | Case | Why |
|-------|------|-----|
| unit (shared module) | default options → no BaseEndpoint, no path-style | plain AWS usage must stay untouched |
| unit (shared module) | WithBaseEndpoint → BaseEndpoint + path-style set | S3-compatibles need explicit endpoint/path-style |
| unit (shared module) | Ping against reachable bucket → nil | happy path |
| unit (shared module) | Ping against missing bucket → error | boot check must actually detect failure |
| unit | none in main-api | wiring is a config guard in `run()` (endpoint set but bucket empty → boot error); covered by the shared-module tests otherwise |
| integration | none yet | no upload feature exists; add floci-backed round-trip test with the first upload endpoint |
| contract/E2E | none yet | same — nothing user-facing consumes storage |

## Inventory admin API (`/admin/items`, `/admin/selling-places`, `/admin/labels`)

**Status:** implemented (Phase 2 of the inventory plan, `docs/plans/inventory-system.md`); E2E pending the Phase 3 frontend.

Single-owner inventory: CRUD for items (with search by name / selling place /
label / Whatnot number), mark-sold, image upload (server-side S3 PUT), and
managed lists for selling places (6 seeded builtins) and labels.

| Level | Case | Why |
|-------|------|-----|
| unit | item create with full intake payload → stored fields round-trip | core intake |
| unit | item create requires name; unknown status/enum rejected | input validation |
| unit | item list filters: name substring, place id, label id, whatnot_number, status | search requirements |
| unit | whatnot_number uniqueness: duplicate non-null → error; multiple NULLs OK | partial unique semantics |
| unit | mark-sold sets status=sold, sold_at/sold_price/sold_place atomically; invalid place → 4xx | sale flow integrity |
| unit | labels/places CRUD: create, list, soft-delete; duplicate name → 409 | managed lists |
| unit | builtin selling-place seed is idempotent (re-run → no duplicates; deleted builtin is NOT resurrected) | boot seed correctness |
| integration | item + images round-trip against floci: upload via API → FileExists in bucket → ItemImage rows ordered by display_order | first upload feature (was deferred in the storage entry above) |
| integration | full flow: create place → create labeled/placed item → search finds it → mark sold → sold fields set | vertical slice |
| unit | item update with `notes: ""` / `whatnot_number: ""` clears the field to NULL | the PWA clears a field by sending an empty string |
| unit | two items both cleared to empty whatnot_number → no unique violation | cleared values must be NULL, not "", or the partial unique index collides |
| contract/E2E | none yet | frontend inventory client ships in Phase 3 |

## Item status flow (draft → listed → sold, archived side exit)

**Status:** implemented

Intake creates items as `draft`. The PWA moves them with `PATCH /admin/items/{id}`
carrying `status`: draft→listed (requires ≥1 selling place, enforced client-side),
listed→draft (unlist), draft/listed→archived, archived→draft (restore).
`listed_at` stamps the current listing and is cleared on unlist; `first_listed_at` is written once on the first listing and never cleared, so the inventory list can show time from listing to sale. Selling goes
through `POST /admin/items/{id}/sold`; afterwards `sold_price_cents` / `sold_at` /
`sold_place_id` stay patchable so a mis-keyed sale can be corrected. The API
validates the status value but does not guard transitions — the flow lives in
the client.

| Level | Case | Why |
|-------|------|-----|
| integration | PATCH status=listed on a draft sets status + stamps listed_at | the core transition |
| integration | re-sending status=listed leaves listed_at unchanged | listing age must not reset on an unrelated save |
| integration | PATCH status=draft clears listed_at | unlisting restarts the clock next time |
| integration | first_listed_at is stamped on the first listing and survives unlist → relist | the list shows time-to-sale from the FIRST listing, which listed_at cannot answer |
| integration | PATCH status=archived from draft and from listed | both side exits |
| integration | PATCH with an invalid status → 422 | enum validation is the only server-side guard |
| integration | PATCH sold_price_cents/sold_at/sold_place_id on a sold item updates the sale; unknown place → 400 | sale corrections without reopening the item |
| integration | PATCH status=sold with no sold_place_id (payload or row) → 400; with one → sold; create with status=sold and no place → 400 | a sold item must name where it sold, or sales-by-platform loses rows |
| integration | hard-deleting a place with sales against it → FK error; the API's soft delete still works and leaves the sale intact | sold_place is RESTRICT, so a sale can never be orphaned by removing a place |
| unit | none | these are DB round-trips; the API package has no unit-level harness |

## Create responses carry a JSON body (`/admin/selling-places`, `/admin/labels`)

**Status:** implemented

Huma emits 204 with the struct's fields as HTTP headers when a response struct
has no `Body` field (the convention noted in the root `AGENTS.md`). Both create
handlers returned the bare output struct, so the PWA's "Add place…" / "Add
label…" rows and the mark-sold Custom… option pushed `undefined` into their
lists and crashed the page on the next render.

| Level | Case | Why |
|-------|------|-----|
| integration | POST selling-place / label → 200 with a JSON body carrying id + name | the regression that broke every inline create |
| integration | the created place is immediately usable as sold_place_id | Custom… in the mark-sold dialog depends on it |
| contract/E2E | none yet | pending an E2E harness for the PWA |
