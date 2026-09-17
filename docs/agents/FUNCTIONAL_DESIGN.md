# How the application works

Functional design for **bright-vintage-finds**: what the product does, who it
does it for, and the rules it enforces. Structure and code layout live in the
`AGENTS.md` files; deployment specifics live in `DEPLOYMENT.md`. This document
is about behaviour.

Where something is deliberately not built yet, it says so. Where the code and
this document disagree, the code is reality — fix the document.

---

## 1. What the product is

A one-person vintage-and-antique resale business, run from a phone.

The owner buys pieces at estate sales, photographs them, records what each one
cost and what it should fetch, sells them (today: live on Whatnot), and wants
to know afterwards what sold, how fast, and for how much. The public website
exists so a buyer can see the current stock and be pushed toward the Whatnot
shop.

Two audiences, one system:

| Audience | Surface | Auth | What they can do |
|---|---|---|---|
| Shoppers (anyone) | Shop front at `/` | none | Browse items currently for sale, search and filter them, view photos, follow to Whatnot |
| The owner (one person) | Inventory PWA at `/inventory*` | Cloudflare Access | Everything: intake, edit, photograph, list, sell, archive, delete |

There is exactly **one** owner. The inventory is not multi-tenant, and nothing
in the product distinguishes users: the API attributes every item to a single
built-in owner row created on first use. Identity comes from Cloudflare
Access at the edge, not from an account system inside the app.

---

## 2. Surfaces

| Surface | Path | Served by | Notes |
|---|---|---|---|
| Shop front | `/` | `public_site` (React, Caddy) | Masthead, hero, catalog grid, "where to buy", footer. One page. |
| Inventory PWA | `/inventory`, `/inventory/new`, `/inventory/item/:id` | same app | Installable (PWA install prompt only — no offline mode) |
| Public catalog API | `/public/*` | `main-api` | Unauthenticated, read-only |
| Admin API | `/admin/*` | `main-api` | Cloudflare Access + in-app assertion check |
| Healthcheck | `/healthz` | platform framework | Skips auth and logging |

The PWA and the admin API share one origin in production (`brightvintagefinds.com`),
because Caddy reverse-proxies `/admin/*` to the API. That keeps the Cloudflare
Access cookie first-party and removes CORS from the picture entirely.

---

## 3. Domain model

### Item

The only entity with rules. Everything else exists to describe items.

| Field | Type | Meaning / rule |
|---|---|---|
| `id` | KSUID string | Sorts by creation time, so "newest first" is `id DESC` and the same value doubles as the catalog paging cursor |
| `name` | string, required | 1–300 characters. The only field intake insists on |
| `description`, `category`, `condition` | nullable strings | Public-facing copy. **The API accepts them; no PWA screen captures them yet** (see §11) |
| `status` | enum | `draft` \| `listed` \| `sold` \| `archived` |
| `acquisition_cost_cents` | nullable int64 | What the owner paid, in USD cents. Never public |
| `purchased_at` | nullable timestamp | When the owner bought it |
| `listing_price_cents` | nullable int64 | Asking price, USD cents. Required by the PWA before listing |
| `length`, `width`, `height` | nullable float | In `measurement_unit` |
| `measurement_unit` | enum | `inch` \| `cm`, defaults to inch. Anything else is rejected |
| `extra_measurements` | nullable string | Free text for dimensions the columns don't cover |
| `weight_lbs`, `weight_oz` | nullable | Shipping weight, split into whole pounds + ounces |
| `notes` | nullable string | Owner's private note. Clearable (see §5.4). Never public |
| `whatnot_number` | nullable string | The item's number in a Whatnot show. Unique **when present**. Clearable. Never public |
| `selling_places` | many-to-many | Where it is offered. At least one is required by the PWA before listing |
| `labels` | many-to-many | Tags for finding it later; also the shop front's tag filter |
| `listed_at` | nullable timestamp | Times the **current** listing. Stamped on the way in, cleared on the way back to draft |
| `first_listed_at` | nullable timestamp | Stamped on the **first** listing and never rewritten. Survives unlist/relist, so "days to sell" stays honest |
| `sold_price_cents`, `sold_at`, `sold_place` | nullable | The sale record |
| `created_at`, `updated_at`, `version`, `deleted_at` | platform mixins | KSUID id, update tracking with optimistic-locking `version`, soft delete |

Money is always **USD cents as an integer**. The UI converts to and from
dollars at the edge; nothing stores a float price.

### Supporting entities

| Entity | Purpose | Rules |
|---|---|---|
| **SellingPlace** | A platform or venue: Whatnot, Local store, Facebook, Mercari, eBay, Poshmark | Names are unique. The six builtins are seeded idempotently at boot; the owner can add custom ones. Soft delete only. A place with sales against it cannot be hard-deleted (`ON DELETE RESTRICT` on the sold edge) |
| **Label** | An item-type tag, e.g. "Glassware" | Unique names, soft delete, owner-created |
| **ItemImage** | One stored photo: bucket, key, filename, content type, display order | The object lives in S3; the row is only a reference. `display_order` decides sequence; the first is the cover |
| **User** | The single built-in owner | Created on first use, well-known row. Not derived from Access headers |

---

## 4. Item lifecycle

```
              ┌──────────── unlist ────────────┐
              ▼                                │
  intake → draft ───── list it ─────────→ listed ───── mark sold ───→ sold
              │                                │                        │
              ├──── archive ──────────→ archived ◀──── archive ─────────┤ (not offered)
              │                                │
              └── delete (never listed) ───────┴── delete (archived) ──→ gone
```

### 4.1 Which moves are offered

The **client owns the flow**; the API validates the value and the bookkeeping
each move implies, not the legality of the transition. Both the inventory list
(long-press action sheet) and the item page offer:

| From | Offered moves | Also offered |
|---|---|---|
| `draft` | List it (needs ≥1 selling place, prompts for price), Archive | Delete, if it was never listed |
| `listed` | Mark sold (needs price, date, place), Unlist → draft, Archive | — |
| `sold` | none — "Sold items stay put. Open it to fix the sale." | — |
| `archived` | Restore to draft | Delete |

### 4.2 Listing timestamps

Decided in one place (`domain.PlanStatusChange` / `PlanCreateStatus`):

- A **first** listing stamps both `listed_at` and `first_listed_at`.
- A **relisting** refreshes `listed_at` only; `first_listed_at` is never rewritten.
- Re-sending the status an item already has touches neither.
- Going back to `draft` clears `listed_at` and leaves `first_listed_at` alone.
- An item created directly as `listed` counts as a first listing.

These drive the two numbers the owner reads on the list: how long an active
item has been out (`listed_at` → now) and how long a sold one took to sell
(`first_listed_at` → `sold_at`).

### 4.3 A sale always names a place

Moving to `sold` is rejected unless a selling place comes from the payload or
is already stored on the item. Sales insight is built on that edge, so the
product refuses to record a sale it cannot attribute.

Marking sold has its own operation (`POST /admin/items/{id}/sold`) because it
always writes the same four things: date, price, place, status. A `PATCH` can
afterwards **correct** `sold_price_cents`, `sold_at` and `sold_place_id` — the
sale is fixable, not re-openable.

### 4.4 Deletion

`domain.CanDelete` allows exactly two cases:

- a **draft that was never listed** (no `first_listed_at`), and
- anything **already archived** — archiving is the deliberate step before disposal.

Everything else is refused with a 409 and a specific message: a sold item is
"a record of the sale and cannot be deleted"; a once-listed item must be
archived first.

Deleting removes things in an order chosen so nothing is ever orphaned in the
bucket: **stored objects first, then photo rows, then the item** (soft delete).
Object clearing covers photos the owner had already removed, not only the ones
still showing.

---

## 5. Owner workflows

### 5.1 Batch intake — `/inventory/new`

Designed for a pile of items on a table, not one careful record at a time.

1. Photos first: **Take photo** (rear camera) or **Library**. Each picture is
   converted and resized in the browser *before* it is staged, so a photo that
   cannot be used fails while the item is still on screen (§6).
2. Fill the fields: name (required), paid, list price, purchased date, Whatnot
   number, dimensions + unit, extra measurements, weight, notes.
3. Tick selling places and labels; either list can be extended inline, and a
   newly created one is auto-ticked.
4. **Save & add another**: the item is created as a `draft`, staged photos
   upload one after another, then the form clears and focus returns to the
   name field. The page stays put, counts "*n* saved this session", and offers
   an **Open** link to the item just saved.
5. Selling places and labels stay ticked between saves — a batch usually shares
   them. Everything else resets.

A photo that fails to upload does **not** fail the item: the item is saved and
the banner explains which file failed and why (HEIC gets its own message).

### 5.2 Working the list — `/inventory`

- Items are grouped by status in flow order: **Draft**, **Active** (the UI name
  for `listed`), **Sold**, **Archived**. Draft and Active are open by default;
  the finished groups start collapsed so old stock does not bury today's work.
- Within a group: newest first.
- Each row shows the cover thumbnail, the name, and whichever of these apply:
  time listed, "sold in *n*d", `WN <number>`, sale price.
- Filters across the top: name search, selling place, label, Whatnot number,
  status.
- **Tap** a row opens the item. **Long press** (500 ms) or right-click opens an
  action sheet with the moves that status allows — listing from here still goes
  through the price dialog, and is blocked with an explanation if the item has
  no selling place yet.

### 5.3 The item page — `/inventory/item/:id`

- Header: name, status pill, and a subtitle that adapts — "Listed 12d ago",
  "Sold 2026-03-04 · 21d to sell", or the Whatnot number.
- Photos strip, cover marked. Photos can be **added while the item is a draft**;
  once listed, the pictures are part of the live listing and the add button is
  gone. There is no delete or reorder yet (§11).
- Details are **read-only until the owner opens the form** with *Edit details*,
  so a phone in a pocket cannot quietly rewrite an item. Whatnot number,
  selling places, labels and notes stay editable outside that form, because
  they change constantly while an item is out.
- Bottom action bar carries the one obvious move for the current status:
  **List it** / **Mark sold** / **Restore to draft**, with the secondary moves
  (unlist, archive, delete) below it.
- **Sold and archived items are records, not forms**: nothing on them is
  editable, and a sold item shows a read-only Sale card (price, date, place,
  Whatnot number).

### 5.4 The two dialogs

| Dialog | Why it exists | Behaviour |
|---|---|---|
| **List it** | A listed item is immediately on the shop front, so it cannot go out without a price | Prompts for the list price, prefilled from the stored one (relist, or a price typed at intake). Confirming sets `status: listed` and the price in one PATCH |
| **Mark sold** | A sale needs price, date and place together | Price defaults to the listing price (most items sell at ask), date defaults to today, place defaults to the item's first selling place. A "Custom…" option creates a real selling place first, so the sale keeps a valid reference |

### 5.5 Clearing a field

`notes` and `whatnot_number` are the only **clearable** fields: the PWA sends an
empty string to mean "clear it", which stores NULL rather than `""`. This
matters for `whatnot_number`, which is unique when present — two `""` rows
would collide. On creation, an empty string means "not given" and is dropped.

Numbers behave differently: an absent number means "leave it alone", so emptying
the list-price box does not clear the stored price (§11).

---

## 6. Photos

| Stage | Behaviour |
|---|---|
| Capture | `accept="image/jpeg,image/png,image/webp"` rather than `image/*`, because iOS hands over the original HEIC for the latter |
| Client-side prepare | Every picture goes through one canvas pass: longest edge capped at 1600 px, JPEG quality 0.82. A 4.5 MB phone photo becomes ~0.58 MB, still sharper than any surface renders it. An already-small file in an accepted format is sent untouched |
| Upload | Multipart `POST /admin/items/{id}/images`, one file per request, sequential |
| Validation | 15 MB cap, `image/jpeg` / `image/png` / `image/webp` only. Size is checked before type, so an oversize GIF reads as too large |
| Storage key | `items/<item-id>/<opaque-id><ext>`, preferring the filename's extension and falling back to the content type's — extensionless phone uploads still land usable |
| Write order | Object first, then the row. A failure leaves an unreferenced object rather than a row pointing at nothing |
| Display order | Next photo goes after the current last. The **first in display order is the cover** |
| Serving | Short-lived presigned GET URLs, valid 15 minutes, minted per request. The public API never exposes bucket or key |

A presign failure surfaces as an error rather than an item with no cover —
silently dropping it would look like a missing photo.

---

## 7. The shop front

- **Hero** with the shop name, tagline and two calls to action: browse the
  collection, or watch on Whatnot. All copy is currently hardcoded in
  `components/shop/config.ts`; none of it is in the database.
- **Catalog grid**: only `listed` items, newest first.
  - Search box (300 ms debounce, server-side name match).
  - Category and tag dropdowns, populated from `/public/filters` — the values
    actually present among listed items, so the shop never offers an empty filter.
  - Cursor paging, 24 per page, with a **Show more** button. A page returns a
    cursor only when another page exists.
  - Changing any filter restarts paging from the first page.
- **Lightbox**: opening a card fetches that item's photos and shows the full set
  plus every public detail. The grid stays one request; photos are fetched on
  demand.
- **Where to buy**: everything sells live on Whatnot; the CTA follows the shop.
- Empty and error states are written as shop copy, not as stack traces
  ("Opening the cabinet…", "Nothing matches that just now.").

### What the public may never see

The public item shape is a deliberate allow-list. An unauthenticated response
carries **id, name, description, category, condition, listing price, dimensions,
labels, image count and a cover URL** — and nothing else. It never carries
acquisition cost, notes, Whatnot numbers, selling places, sold data, or storage
keys and buckets. Photos of an item that is not listed are private: the photo
route 404s unless the item is currently listed.

---

## 8. API contract

### Admin (`/admin/*`, Cloudflare Access)

| Method | Path | Does |
|---|---|---|
| `POST` | `/admin/items` | Create an item (defaults to `draft`) |
| `GET` | `/admin/items` | List live items, newest first; filters `query`, `place_id`, `label_id`, `whatnot_number`, `status` |
| `GET` | `/admin/items/{id}` | One live item |
| `PATCH` | `/admin/items/{id}` | Partial update, including status moves and sale corrections |
| `DELETE` | `/admin/items/{id}` | Delete if the rules allow (§4.4) |
| `POST` | `/admin/items/{id}/sold` | Record a sale: date, price, place |
| `GET` | `/admin/items/{id}/images` | Photos with fresh view URLs |
| `POST` | `/admin/items/{id}/images` | Upload one photo (multipart) |
| `GET`/`POST` | `/admin/selling-places` | List / add |
| `DELETE` | `/admin/selling-places/{id}` | Soft delete |
| `GET`/`POST` | `/admin/labels` | List / add |
| `DELETE` | `/admin/labels/{id}` | Soft delete |

The image routes are registered **only when object storage is configured**.

### Public (`/public/*`, open)

| Method | Path | Does |
|---|---|---|
| `GET` | `/public/items` | Listed items, cursor-paginated (`limit` 1–60, default 24; `cursor`, `query`, `category`, `label`) |
| `GET` | `/public/filters` | Categories and labels present among listed items |
| `GET` | `/public/items/{id}/images` | Photos of a listed item; 404 otherwise |

### Validation and errors

Error **wording is written in the domain**, so a refactor cannot silently change
the API contract; the HTTP layer only picks the status code:

| Domain kind | HTTP | Example |
|---|---|---|
| invalid | 400 | `invalid status "pending"`, `measurement_unit must be inch or cm`, `unknown selling_place_ids`, `a sold item needs sold_place_id` |
| not found | 404 | `item not found` |
| conflict | 409 | `sold items are a record of the sale and cannot be deleted`; duplicate Whatnot number |
| too large | 413 | `image exceeds 15 MB` |
| unavailable | 503 | `object storage is not configured` |
| anything else | 500 | detail not leaked |

Ordering matters and is tested: an unparseable status is reported **before**
unknown edge ids, so a doubly-invalid payload always fails the same way.

---

## 9. Access, privacy and trust boundaries

- Cloudflare Access authenticates the owner at the edge and injects a signed
  assertion header. The API **independently verifies** it — signature against
  the team JWKS, audience against the app's AUD tag — so the backend never
  trusts the edge blindly.
- One Access application covers all three admin destinations (`api.…/admin`,
  `site/admin`, `site/inventory`) under one policy, therefore one AUD, which is
  what the API is configured to accept.
- Unconfigured server: in **development** admin paths are open (there is no
  Cloudflare edge in front of it); in **production** they fail closed.
- The shop front and `/public/*` are deliberately open and must stay safe for
  anyone to read (§7).
- Trust boundaries that get real validation: image uploads (size, then type),
  every status and unit string, every referenced selling place and label id,
  and page-size clamping on the public catalog.
- Secrets (`MAIN_DB_URL`, `REDIS_URL`) carry `json:"-"` so the startup config
  dump never logs them.

---

## 10. Degraded and partial configurations

The product is built to boot in pieces rather than refuse to start:

| Missing | Behaviour |
|---|---|
| No database (`MAIN_DB_URL` unset, development only) | Server boots, only platform routes are served. No inventory, no catalog |
| No object storage (`S3_BASE_ENDPOINT` unset) | Everything works except photos: the upload/list image routes are never registered, the admin list still reports `image_count`, the public catalog reports `0` and no covers, and the photo use cases answer 503 rather than dereferencing a nil store |
| Storage misconfigured | Boot pings (HeadBucket) the upload bucket and **fails fast**, rather than discovering it on the owner's first upload |
| Broken selling-place seed | Surfaces at deploy time: the builtin places are seeded at boot, fail-fast, not when the intake page first loads |

### Configuration

| Variable | Purpose |
|---|---|
| `ENV`, `SERVER_PORT` | Required. `development` \| `production` |
| `MAIN_DB_URL` | Postgres DSN. Required in production |
| `S3_BASE_ENDPOINT`, `S3_UPLOAD_BUCKET` | Object storage. Bucket required when the endpoint is set |
| `S3_PUBLIC_ENDPOINT` | Rewrites an internal presign host to its public form — exactly once, and only when both endpoints are set |
| `CF_ACCESS_TEAM_DOMAIN`, `CF_ACCESS_AUD` | Admin guard. Set both or neither |
| `AWS_REGION` | AWS client region |
| `REDIS_URL` | Declared, not yet used |

---

## 11. Deliberate behaviours and known gaps

These look like bugs and are documented decisions — do not "fix" them without a
decision:

- **The update path finds soft-deleted rows; the delete path does not.** The two
  handlers queried differently before the hexagonal refactor, and that
  difference was preserved.
- **The admin list loads photos per item; the catalog batches a whole page into
  one query.** The public page can be large; the admin list is owner-only.
- **The admin list reports `image_count` without storage configured; the public
  catalog reports `0`** — it never loads the photo rows at all.
- **`POST /admin/items` rejects `status: "sold"` without `sold_place_id` but
  never stores the place.** Carried over from the pre-refactor handler; likely a
  bug, left alone.

Genuine gaps, in rough order of how much they cost the owner:

1. **No screen captures `description`, `category` or `condition`.** The API,
   database and public response all support them, and the shop front filters by
   category — but nothing can set one, so the category filter can never appear.
   This is the biggest gap between what the shop front can do and what it does.
2. **No sales insight.** Sale figures are stored on the item; there is no
   dashboard, no aggregate, no separate sales table. "Gains insight into their
   own sales" is stated intent, not shipped behaviour.
3. **Photos cannot be deleted or reordered.** The cover is whichever photo was
   uploaded first, and only draft items accept new ones.
4. **Clearing a number is impossible.** Emptying the list-price or paid box
   leaves the stored value alone; only `notes` and `whatnot_number` have a clear
   convention.
5. **No sales portal**, no multi-user support, no customer accounts, no checkout —
   selling happens on Whatnot, and the site points at it.
6. **No offline mode.** The PWA is install-only; every screen needs the network.
7. **No graceful shutdown** in the API yet (marked in code).
8. Shop-front copy (name, tagline, intro, Whatnot handle) is **hardcoded**, not
   editable by the owner.

---

## 12. Where the rules actually live

For anyone changing behaviour — the rule belongs in exactly one of these:

| Rule | Home |
|---|---|
| Status parsing, listing timestamps, sold-needs-a-place, deletion rules, upload validation, object keys, cover selection, page-size clamping, error wording | `backend/main-api/internal/app/domain` |
| Use cases that sequence those rules over storage | package `app` (`internal/app/{app,items,catalog,images}.go`) |
| Which moves are *offered*, when a price is demanded, read-only vs editable, batch intake behaviour | the PWA (`frontend/public_site/src/pages`) |
| Wire shape, status codes, public field allow-list | `backend/main-api/api/routes` |
| Row mapping, query transcription | `internal/app/adapters/ent` |

The client owns the flow; the domain owns the consequences. If a rule protects
data (a sale record, a unique column, a trust boundary), it belongs in the
domain even when the client also enforces it.
