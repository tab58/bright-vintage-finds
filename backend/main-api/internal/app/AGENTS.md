# internal/app — application core

Ports and adapters (hexagonal). The rules live here; the HTTP layer in
`api/` only translates the wire format. Plan and rationale:
`docs/plans/hexagonal-main-api.md`.

## Layout

| Path | Holds | May import |
|---|---|---|
| `domain/` | Types (`Item`, `ItemImage`, `Label`, `SellingPlace`), parsers, status and deletion rules, `Error` with its `Kind` | stdlib only |
| `ports/` | Repository and storage interfaces the use cases need | stdlib, `domain` |
| `ports/fakes/` | In-memory port implementations for the use-case tests | stdlib, `domain`, `ports` |
| `adapters/ent/` | Repository ports over the Ent client; the only place that maps rows to domain types | `db/generated`, `domain`, `ports` |
| `adapters/s3/` | `ImageStore` over the shared `aws_s3` client | `aws_s3`, `domain`, `ports` |
| `app.go` | `Application` (the one root object), `Config`, `New`, and the label/selling-place use cases | stdlib, `domain`, `ports` |
| `items.go` | Item use cases: `CreateItem`, `GetItem`, `ListItems`, `UpdateItem`, `MarkItemSold`, `DeleteItem` | stdlib, `domain`, `ports` |
| `catalog.go` | The public catalog projection: `Catalog`, `CatalogFacets`, `CatalogImages` | stdlib, `domain`, `ports` |
| `images.go` | Photo use cases: `ListItemImages`, `UploadItemImage` | stdlib, `domain`, `ports` |
| `views.go` | The read models and wire-shaped inputs (`ItemView`, `CatalogPage`, `ItemInput`, …) | stdlib, `domain` |

The import direction is one-way. **`domain` imports nothing but the standard
library** — no Ent, no huma, no `net/http`. If a change wants to add one of
those, the logic belongs in a use case or an adapter instead.

Package `app` holds the use cases and nothing else: it imports `domain` and
`ports`, never `adapters/*` or `db/`. The composition root is
`cmd/app/main.go`, which builds the Ent repositories and the S3 store and
hands them to `app.New(app.Config{...})` as ports. That is also how the
integration tests in `api/routes` wire a real database, and how the unit tests
here wire `ports/fakes`.

## Rules that live here, not in handlers

- **The PATCH tri-state.** `domain.ClearableString` distinguishes *absent*
  (leave), *empty string* (clear) and *value* (set). Only `notes` and
  `whatnot_number` are clearable. A cleared column must become NULL, never
  `""`: `whatnot_number` is unique when present, so two `""` rows collide.
- **Listing timestamps.** `domain.PlanStatusChange` decides them. A first
  listing stamps `listed_at` and `first_listed_at`; a relisting refreshes only
  `listed_at`; re-sending the current status touches neither; going back to
  draft clears `listed_at`. `first_listed_at` is written once and never
  rewritten.
- **Sold needs a place.** From the payload or already stored; sales insight is
  built on that edge.
- **Deletion.** `domain.CanDelete`: never-listed drafts and archived items
  only. Deleting removes stored objects first, then photo rows, then the item.
- **Error wording.** The message a client sees is written in the domain, not
  at the edge, so a refactor cannot silently change the API contract.
  `api/routes/errors.go` only picks the status code from `Kind`.

## Behaviour preserved deliberately

These look like inconsistencies and are not to be "fixed" without a decision:

- `ItemRepository.StatusState` (the update path) finds soft-deleted rows;
  `LiveStatusState` (the delete path) does not. The two handlers queried
  differently before the move.
- `ListForItemIncludingDeleted` exists because item deletion clears objects
  for photos the owner had already removed.
- The admin item list reports `image_count` without object storage; the public
  catalog reports `0`, because it never loads the photo rows at all.
- The admin list loads photos per item; the catalog batches a whole page into
  one query. The public page can be large; the admin list is owner-only.
- `POST /admin/items` rejects `status: "sold"` without `sold_place_id` but
  never stores the place. Carried over from the old handler; likely a bug,
  left alone by the refactor.

## Testing

`domain` and the use cases in `app` are unit-tested against the fakes — no
Postgres, no S3 (`go test ./internal/app/...`). `adapters/ent` has no unit tests; it is
covered by the integration tests in `api/routes/`, which need the local Docker stack
(`task up`, then `task run-integration-tests`). Which rule each test proves — and
which rules nothing proves yet — is tracked in
`backend/main-api/testing/REGISTRY.md`.
