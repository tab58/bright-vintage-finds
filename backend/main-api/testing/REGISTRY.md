# main-api business rule registry

Every business rule this product enforces, where it is enforced, and the test
that proves it. This is the coverage record for `backend/main-api` **and** for
the rules the inventory PWA owns, because a rule enforced only in the client is
still a rule — and, today, an untested one.

Supersedes `backend/main-api/test-cases.md` (folded in and deleted, 2026-09-16).
Behaviour itself is described in [`docs/agents/FUNCTIONAL_DESIGN.md`](../../../docs/agents/FUNCTIONAL_DESIGN.md);
this file does not restate it, it enumerates and tracks it.

## Levels

| Level | What it means here | How to run |
|---|---|---|
| **unit** | `go test`, in-memory ports from `internal/app/ports/fakes`, no Docker, no network | `task run-unit-tests` |
| **integration** | `//go:build integration` in `api/routes/`, real Postgres (local Docker stack), API driven through `humatest`; object storage is a local stub, not floci | `task up`, then `task run-integration-tests` |
| **E2E** | Nothing exists. No browser suite, no API-level flow suite, and the frontend has **no test tooling at all** | — |

## Coverage snapshot (2026-09-16)

96 test functions across 14 files (95 distinct names — `TestSellingPlaceCRUD`
exists in both the unit and integration suites): `internal/app/domain` 17,
package `app` 40, `ports/fakes` 11, `adapters/s3` 4, `internal/cfaccess` 3,
`api/routes` integration 21.

| Area | Rules | covered | partial | gap | known-bug |
|---|---|---|---|---|---|
| `BR-ITEM` — item data | 15 | 12 | 2 | 1 | — |
| `BR-STATUS` — lifecycle | 8 | 8 | — | — | — |
| `BR-SALE` — sales | 5 | 4 | — | — | 1 |
| `BR-DEL` — deletion | 7 | 7 | — | — | — |
| `BR-PHOTO` — photos | 16 | 14 | 1 | 1 | — |
| `BR-CAT` — public catalog | 11 | 11 | — | — | — |
| `BR-TAX` — places & labels | 6 | 4 | 2 | — | — |
| `BR-SEC` — access | 9 | 5 | 1 | 3 | — |
| `BR-OPS` — boot & degraded modes | 8 | 1 | 4 | 3 | — |
| `BR-FLOW` — PWA-enforced | 11 | — | — | 11 | — |
| **Total** | **96** | **66** | **10** | **19** | **1** |

Every test function maps to at least one rule except two kinds of scaffolding:
the `ports/fakes` suite, which proves the **in-memory fakes reproduce Postgres
behaviour** (unique Whatnot numbers, soft-delete visibility, cursor paging,
display ordering, idempotent seeding) so a green unit test is not a false
negative — those are cited as *fake fidelity* against the rule they protect —
and `fakes/fakes_test.go::TestFakeErrorInjection`, which exercises the test
helper itself and backs no product rule.

The shape of the gaps: the **domain and the use cases are thoroughly covered**;
what is not covered is everything at the **edges** — process boot and config,
the multipart upload endpoint, and every rule the client enforces.

## Conventions

- **ID**: `BR-<AREA>-<NNN>`, sequential within an area, never reused. A rule that
  stops being true is struck through with a date, not deleted and not renumbered.
- **Enforced in**: `domain` · `app` · `routes` · `ent` · `adapters/s3` ·
  `cfaccess` · `cmd` · `PWA`. More than one means defence in depth, and the
  registry names the authoritative one first.
- **Status**:
  - `covered` — a test asserts this rule at the level the rule needs.
  - `partial` — proven at one level but not where it can actually break (e.g.
    proven against fakes, never against Postgres; or the DB constraint is proven
    but the HTTP status mapping is not).
  - `gap` — nothing asserts it.
  - `known-bug` — the implemented behaviour diverges from the intended rule, on
    purpose, pending a decision.
- **Proof**: `file::TestFunc`, tagged `[u]` unit or `[i]` integration. Paths are
  relative to `backend/main-api/`. The mapping lives here only — Go test
  functions carry no rule-ID comments.

---

## Rules

### BR-ITEM — item data

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-ITEM-001 | An item's name is required, 1–300 characters | routes (`items.go` schema), PWA | gap | Nothing posts an empty or 301-char name |
| BR-ITEM-002 | Every item belongs to the single builtin owner, resolved once per create | app `items.go`, ent `owner_repo.go` | covered | `app/items_test.go::TestCreateAttributesTheItemToTheBuiltinOwner` [u]; `routes/api_integration_test.go::TestEnsureOwnerIdempotent` [i] |
| BR-ITEM-003 | An item with no status given is created as `draft`, with no listing timestamps | app `items.go`, ent schema default | covered | `app/items_test.go::TestCreateStatusRules` [u] |
| BR-ITEM-004 | Money is stored as USD cents (int64); no float prices anywhere | domain, ent schema | partial | Exercised throughout (`TestMarkSoldRecordsTheSale` [u], `TestSaleCorrections` [i]) but never asserted as a rule |
| BR-ITEM-005 | `measurement_unit` is `inch` or `cm`; unknown and empty are rejected; absent leaves the stored value | domain `item.go`, app `items.go` | covered | `domain/item_test.go::TestParseMeasurementUnit` [u]; `app/items_test.go::TestMeasurementUnitIsParsedFromTheWire` [u] |
| BR-ITEM-006 | `notes` and `whatnot_number` are tri-state: absent leaves, `""` clears to NULL, a value sets | domain `item_patch.go`, app `items.go` | covered | `domain/item_patch_test.go::TestClearable` [u]; `app/items_test.go::TestUpdateAppliesTheClearableTriState` [u]; `fakes/fakes_test.go::TestItemRepoUpdateAppliesTheTriState` [u]; `routes/api_integration_test.go::TestClearFieldsWithEmptyString` [i] |
| BR-ITEM-007 | On create, a blank `notes`/`whatnot_number` is dropped, never stored as `""` | domain `ClearableString.Stored`, app | covered | `domain/item_patch_test.go::TestClearableStored` [u]; `app/items_test.go::TestCreateDropsBlankNotesAndWhatnotNumber` [u] |
| BR-ITEM-008 | `whatnot_number` is unique when present; any number of NULLs coexist | ent (partial unique index) | covered | `routes/api_integration_test.go::TestWhatnotNumberUniqueWhenSet` [i]; fake fidelity: `fakes/fakes_test.go::TestItemRepoWhatnotNumberIsUnique` [u] |
| BR-ITEM-009 | Unknown `selling_place_ids` / `label_ids` are rejected, naming the offending list | app `items.go` | covered | `app/items_test.go::TestCreateRejectsUnknownEdges` [u] |
| BR-ITEM-010 | An unparseable status is reported before unknown edge ids | app `items.go` (ordering) | covered | `app/items_test.go::TestCreateRejectsAnUnparseableStatusBeforeCheckingEdges` [u] |
| BR-ITEM-011 | On update, a nil edge list leaves the set alone; a non-nil one replaces it wholesale | app `items.go` | covered | `app/items_test.go::TestUpdateReplacesEdgeSetsOnlyWhenSent` [u]; `fakes/fakes_test.go::TestItemRepoUpdateReplacesEdgeSetsWholesale` [u] |
| BR-ITEM-012 | The admin list returns live items newest first, narrowed by name / place / label / Whatnot number / status; an invalid status is rejected | app `items.go`, ent `item_repo.go` | partial | `app/items_test.go::TestListFiltersAndRejectsABadStatus` [u]; `fakes/fakes_test.go::TestItemRepoListIsNewestFirstAndFiltered` [u]. Against Postgres the filters are only exercised as hand-written Ent queries (`TestItemLifecycleWithEdgesAndSearch` [i]), **not** through `GET /admin/items`, so the adapter's filter transcription is unproven |
| BR-ITEM-013 | A soft-deleted item disappears from every read path | ent, app | covered | `fakes/fakes_test.go::TestItemRepoSoftDeleteHidesFromReads` [u]; `routes/api_integration_test.go::TestDeleteOnlyNeverListedDrafts` [i]; `routes/public_catalog_integration_test.go::TestPublicItemsOnlyShowsListed` [i] |
| BR-ITEM-014 | A response's `first_listed_at` falls back to `listed_at` for pre-column rows; the deletion rules read the raw column instead | domain `item.go` | covered | `domain/item_test.go::TestEffectiveFirstListedAt` [u] |
| BR-ITEM-015 | An item response carries both names and ids for selling places and labels | domain `item.go`, routes | covered | `domain/item_test.go::TestItemEdgeAccessors` [u]; `routes/api_integration_test.go::TestItemLifecycleWithEdgesAndSearch` [i] |

### BR-STATUS — lifecycle and listing timestamps

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-STATUS-001 | Status is one of `draft`/`listed`/`sold`/`archived`; anything else is rejected with `invalid status "x"` | domain `item.go` | covered | `domain/item_test.go::TestParseStatus` [u]; `app/items_test.go::TestCreateStatusRules` [u]; `routes/api_integration_test.go::TestItemStatusFlow` [i] |
| BR-STATUS-002 | A first listing stamps both `listed_at` and `first_listed_at` | domain `item_rules.go` | covered | `domain/item_rules_test.go::TestPlanStatusChange` [u]; `app/items_test.go::TestUpdateStatusTimestamps` [u]; `routes/api_integration_test.go::TestItemStatusFlow` [i] |
| BR-STATUS-003 | A relisting refreshes `listed_at` only; `first_listed_at` is never rewritten | domain `item_rules.go` | covered | `domain/item_rules_test.go::TestPlanStatusChange` [u]; `app/items_test.go::TestUpdateFirstListedAtSurvivesRelisting` [u]; `routes/api_integration_test.go::TestFirstListedAtSurvivesRelist` [i] |
| BR-STATUS-004 | Re-sending the status an item already has touches neither timestamp | domain `item_rules.go` | covered | `domain/item_rules_test.go::TestPlanStatusChange` [u]; `routes/api_integration_test.go::TestItemStatusFlow` [i] |
| BR-STATUS-005 | Going back to `draft` clears `listed_at` and leaves `first_listed_at` | domain `item_rules.go` | covered | `domain/item_rules_test.go::TestPlanStatusChange` [u]; `app/items_test.go::TestUpdateStatusTimestamps` [u]; `routes/api_integration_test.go::TestItemStatusFlow` [i] |
| BR-STATUS-006 | Archiving touches no timestamps, from either draft or listed | domain `item_rules.go` | covered | `domain/item_rules_test.go::TestPlanStatusChange` [u]; `app/items_test.go::TestUpdateStatusTimestamps` [u]; `routes/api_integration_test.go::TestItemStatusFlow` [i] |
| BR-STATUS-007 | An item created directly as `listed` counts as a first listing | domain `PlanCreateStatus` | covered | `domain/item_rules_test.go::TestPlanCreateStatus` [u]; `app/items_test.go::TestCreateStatusRules` [u] |
| BR-STATUS-008 | The API validates the status *value* but not the *transition* — the flow is the client's (see `BR-FLOW`) | routes, app | covered | `routes/api_integration_test.go::TestItemStatusFlow` [i] drives every move the PWA offers, plus a rejected value |

### BR-SALE — recording and correcting a sale

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-SALE-001 | Moving to `sold` is refused unless a selling place comes from the payload or is already stored | domain `item_rules.go` | covered | `domain/item_rules_test.go::TestPlanStatusChange` + `TestPlanCreateStatus` [u]; `app/items_test.go::TestUpdateToSoldNeedsAPlaceFromEitherSide` [u]; `routes/api_integration_test.go::TestSoldRequiresPlace` [i] |
| BR-SALE-002 | `POST /admin/items/{id}/sold` writes status, date, price and place together and touches nothing else | app `items.go`, routes | covered | `app/items_test.go::TestMarkSoldRecordsTheSale` [u]; `routes/api_integration_test.go::TestSaleCorrections` [i] |
| BR-SALE-003 | An unknown `sold_place_id` is rejected with `unknown sold_place_id` | app `items.go` | covered | `app/items_test.go::TestUpdateRejectsAnUnknownSoldPlace` [u]; `routes/api_integration_test.go::TestSaleCorrections` [i] |
| BR-SALE-004 | A recorded sale stays correctable by PATCH (price, date, place) without reopening the item | routes, app | covered | `routes/api_integration_test.go::TestSaleCorrections` [i] |
| BR-SALE-005 | Creating an item with `status: sold` and a `sold_place_id` should store that place | app `items.go` | **known-bug** | The place is validated for presence but never stored — carried over from the pre-hexagonal handler. `routes/api_integration_test.go::TestSoldRequiresPlace` [i] locks in the 400 for a missing place; **nothing locks in the divergence itself** |

### BR-DEL — deletion

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-DEL-001 | A never-listed draft is deletable | domain `CanDelete` | covered | `domain/item_rules_test.go::TestCanDelete` [u]; `app/items_test.go::TestDeleteEnforcesTheRulesAndClearsPhotos` [u]; `routes/api_integration_test.go::TestDeleteOnlyNeverListedDrafts` [i] |
| BR-DEL-002 | An archived item is deletable — archiving is the deliberate step before disposal | domain `CanDelete` | covered | same three as BR-DEL-001 |
| BR-DEL-003 | A once-listed draft or a live listing is refused with "archive it before deleting" (409) | domain `CanDelete`, routes `errors.go` | covered | `domain/item_rules_test.go::TestCanDelete` [u]; `routes/api_integration_test.go::TestDeleteOnlyNeverListedDrafts` [i] |
| BR-DEL-004 | A sold item is never deletable, archived or not (409) | domain `CanDelete` | covered | `domain/item_rules_test.go::TestCanDelete` [u]; `routes/api_integration_test.go::TestDeleteOnlyNeverListedDrafts` [i] |
| BR-DEL-005 | Deleting removes stored objects first, then photo rows, then the item | app `items.go` | covered | `app/items_test.go::TestDeleteEnforcesTheRulesAndClearsPhotos` [u]; `routes/api_integration_test.go::TestDeleteDraftRemovesImages` [i] |
| BR-DEL-006 | Object clearing covers photos the owner had already removed (soft-deleted rows) | app `items.go`, ent `ListForItemIncludingDeleted` | covered | `app/items_test.go::TestDeleteEnforcesTheRulesAndClearsPhotos` [u]; `fakes/fakes_test.go::TestItemImageRepoDeletedRowsStayVisibleToTheDeletePath` [u] |
| BR-DEL-007 | Without object storage the delete still removes the rows and the item | app `items.go` | covered | `app/items_test.go::TestDeleteWithoutStorageStillRemovesTheRows` [u] |

### BR-PHOTO — photos

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-PHOTO-001 | An upload over 15 MB is rejected | domain `image.go` | covered | `domain/image_test.go::TestValidateUpload` [u]; `app/images_test.go::TestUploadRejectsBadPayloads` [u] |
| BR-PHOTO-002 | Only `image/jpeg`, `image/png`, `image/webp` are accepted, matched case-insensitively | domain `image.go`, routes (multipart contentType) | covered | `domain/image_test.go::TestValidateUpload` [u]; `app/images_test.go::TestUploadRejectsBadPayloads` [u] |
| BR-PHOTO-003 | Size is checked before type, so an oversize unsupported file reads as too large | domain `image.go` | covered | `domain/image_test.go::TestValidateUpload` [u] |
| BR-PHOTO-004 | The object key is `items/<item-id>/<opaque-id><ext>`; the filename's extension wins, the content type's is the fallback | domain `ObjectKey` | covered | `domain/image_test.go::TestObjectKey` [u]; `app/images_test.go::TestUploadStoresTheObjectThenTheRow` [u] |
| BR-PHOTO-005 | The object is written before the row, so a failure leaves no row pointing at nothing | app `images.go` | covered | `app/images_test.go::TestUploadStoresTheObjectThenTheRow` [u] |
| BR-PHOTO-006 | An upload against a missing or dead item is refused before storage is spent | app `images.go` | covered | `app/images_test.go::TestUploadNeedsTheItemToExist` [u] |
| BR-PHOTO-007 | A new photo takes the next display order; lists come back in display order | app `images.go`, ent | covered | `app/images_test.go::TestUploadStoresTheObjectThenTheRow` + `TestListImagesReturnsFreshURLsInDisplayOrder` [u]; `fakes/fakes_test.go::TestItemImageRepoOrdersByDisplayOrder` [u] |
| BR-PHOTO-008 | The cover is the first photo in display order, not the first uploaded | domain `Cover`, app | covered | `domain/image_test.go::TestCover` [u]; `app/items_test.go::TestViewCarriesPhotoCountAndCover` [u]; `routes/api_integration_test.go::TestListCarriesCoverImageURL` [i] |
| BR-PHOTO-009 | View URLs are presigned and short-lived (15 minutes) | domain `PresignTTL`, `adapters/s3` | partial | URLs are asserted present and correct (`app/images_test.go::TestListImagesReturnsFreshURLsInDisplayOrder` [u]) but **no test asserts the TTL value** reaching the presigner |
| BR-PHOTO-010 | A presign failure surfaces as an error, never as an item with no cover | app, `adapters/s3` | covered | `app/images_test.go::TestPresignFailureIsReportedNotSwallowed` [u]; `adapters/s3/store_test.go::TestViewURLReportsPresignFailureAsInternal` [u] |
| BR-PHOTO-011 | An internal presign host is rewritten to its public form exactly once, and only when both endpoints are configured | `adapters/s3/store.go` | covered | `adapters/s3/store_test.go::TestViewURLRewritesTheEndpoint` [u] |
| BR-PHOTO-012 | Without object storage, listing and uploading photos report 503 rather than dereferencing a nil store | app `images.go` | covered | `app/images_test.go::TestUploadWithoutStorageIsUnavailable` + `TestListItemImagesWithoutStorageIsUnavailable` [u] |
| BR-PHOTO-013 | An empty filename is stored as NULL, never as `""` | domain `FilenameOrNil` | covered | `domain/image_test.go::TestFilenameOrNil` [u] |
| BR-PHOTO-014 | `POST /admin/items/{id}/images` accepts a real multipart photo and buffers it in memory (the production image is `FROM scratch` and has no `/tmp`) | routes `item_images.go` | gap | The image routes are **never registered** in the integration harness — `RegisterItemImageRoutes` appears in no test. The 422 "cannot read multipart form" bug this rule exists for could recur undetected |
| BR-PHOTO-015 | New objects go to the configured upload bucket | `adapters/s3/store.go` | covered | `adapters/s3/store_test.go::TestBucketIsTheConfiguredUploadTarget` [u]; `app/images_test.go::TestUploadStoresTheObjectThenTheRow` [u] |
| BR-PHOTO-016 | Storage failures surface as internal domain errors, never as raw AWS errors | `adapters/s3/store.go` | covered | `adapters/s3/store_test.go::TestUploadAndDeletePassThroughAndWrapFailures` [u] |

### BR-CAT — public catalog

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-CAT-001 | Only `listed`, non-deleted items are public | app `catalog.go`, ent | covered | `app/catalog_test.go::TestCatalogShowsOnlyListedItems` [u]; `routes/public_catalog_integration_test.go::TestPublicItemsOnlyShowsListed` [i] |
| BR-CAT-002 | The public item payload carries no acquisition cost, purchase date, notes, Whatnot number, selling places, sold data or status | routes `public_items.go` | covered | `routes/public_catalog_integration_test.go::TestPublicItemsOnlyShowsListed` [i] (asserts on raw JSON keys) |
| BR-CAT-003 | Page size defaults to 24 and is capped at 60 | domain `ClampPageSize`, routes schema | covered | `domain/item_patch_test.go::TestClampPageSize` [u] |
| BR-CAT-004 | Cursor paging walks newest-first with no repeats and no gaps | app `catalog.go`, ent | covered | `app/catalog_test.go::TestCatalogPagesWithACursor` [u]; `fakes/fakes_test.go::TestItemRepoCatalogPagingWalksTheCursor` [u]; `routes/public_catalog_integration_test.go::TestPublicItemsPagination` [i] |
| BR-CAT-005 | `next_cursor` is present only when another page exists — an exactly-full page has none | app `catalog.go` | covered | `app/catalog_test.go::TestCatalogPageExactlyFullHasNoCursor` [u]; `routes/public_catalog_integration_test.go::TestPublicItemsPagination` [i] |
| BR-CAT-006 | `query`, `category` and `label` filter server-side | app, ent | covered | `app/catalog_test.go::TestCatalogFiltersByCategoryAndLabel` [u]; `routes/public_catalog_integration_test.go::TestPublicItemsFilters` [i] |
| BR-CAT-007 | `/public/filters` lists only the categories and labels present among listed items | app `catalog.go`, ent | covered | `app/catalog_test.go::TestCatalogFacetsComeFromListedItemsOnly` [u]; `fakes/fakes_test.go::TestItemRepoListedFacetsCoverListedItemsOnly` [u]; `routes/public_catalog_integration_test.go::TestPublicFiltersFromListedItemsOnly` [i] |
| BR-CAT-008 | Photos of an item that is not listed are private: the public photo route 404s for draft, sold, archived and unknown ids | app `catalog.go` | covered | `app/catalog_test.go::TestCatalogImagesAreGatedByStatus` [u]; `routes/public_catalog_integration_test.go::TestPublicItemImagesGatedByStatus` [i] |
| BR-CAT-009 | The public image payload carries no `upload_key` or `upload_bucket` | routes `public_items.go` | covered | `routes/public_catalog_integration_test.go::TestPublicItemImagesGatedByStatus` [i] |
| BR-CAT-010 | A catalog page loads its covers in one batched query, not one per item | app `catalog.go` | covered | `app/catalog_test.go::TestCatalogBatchesCoverLookups` [u] |
| BR-CAT-011 | Without storage the catalog reports `0` images and no covers, while the admin list still reports its count — a deliberate divergence | app `catalog.go` vs `items.go` | covered | `app/catalog_test.go::TestCatalogWithoutStorageReportsNoPhotos` + `TestCatalogImagesWithoutStorageIsEmptyNotNil` [u]; `app/items_test.go::TestViewWithoutStorageCountsPhotosButHasNoCover` [u]; `routes/api_integration_test.go::TestListWithoutStorageHasNoCover` [i] |

### BR-TAX — selling places and labels

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-TAX-001 | The six builtin selling places are seeded at boot, idempotently | domain `BuiltinSellingPlaces`, app, ent | covered | `app/app_test.go::TestSeedBuiltinPlacesIsIdempotent` [u]; `fakes/fakes_test.go::TestSellingPlaceEnsureBuiltinsIsIdempotent` [u]; `routes/api_integration_test.go::TestSeedBuiltinSellingPlacesIdempotent` [i] |
| BR-TAX-002 | A builtin the owner soft-deleted is not resurrected by the next boot seed | ent `selling_place_repo.go` | covered | `fakes/fakes_test.go::TestSellingPlaceEnsureBuiltinsIsIdempotent` [u]; `routes/api_integration_test.go::TestSeedBuiltinSellingPlacesIdempotent` [i] |
| BR-TAX-003 | Place and label names are unique; a duplicate is refused | ent schema | partial | The DB constraint is proven (`routes/api_integration_test.go::TestSellingPlaceCRUD` [i], driving Ent directly). **The API's 409 mapping for a duplicate is never asserted** |
| BR-TAX-004 | Deleting a place or label is a soft delete; an unknown or empty id errors | app `app.go`, ent | covered | `app/app_test.go::TestLabelCRUD` + `TestSellingPlaceCRUD` [u]; `routes/api_integration_test.go::TestSellingPlaceCRUD` [i] |
| BR-TAX-005 | A selling place with sales against it cannot be hard-deleted; the API's soft delete leaves the sale intact | ent (`ON DELETE RESTRICT`) | covered | `routes/api_integration_test.go::TestSoldPlaceCannotBeHardDeleted` [i] |
| BR-TAX-006 | Create responses carry a JSON body — a bodyless huma response 204s and breaks every inline "Add place/label" | routes | partial | `routes/api_integration_test.go::TestCreateReturnsJSONBody` [i] covers places and labels; the item and sold endpoints are not re-checked for the same trap |

### BR-SEC — access and trust boundaries

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-SEC-001 | When configured, `/admin/*` requires a valid Cloudflare Access assertion, verified in-app against the team JWKS | cfaccess | covered | `cfaccess/cfaccess_test.go::TestConfiguredVerification` [u] |
| BR-SEC-002 | Expired, wrong-audience, unknown-key, unsigned-garbage and expiry-less assertions are all 401 | cfaccess | covered | `cfaccess/cfaccess_test.go::TestConfiguredVerification` [u] |
| BR-SEC-003 | Non-admin paths pass through untouched, and `/administrata` is not an admin path | cfaccess | covered | `cfaccess/cfaccess_test.go::TestConfiguredVerification` + `TestUnconfiguredModes` [u] |
| BR-SEC-004 | Unconfigured + development leaves `/admin` open; unconfigured + production fails closed, including the bare `/admin` path | cfaccess | covered | `cfaccess/cfaccess_test.go::TestUnconfiguredModes` [u] |
| BR-SEC-005 | Half-configured (team domain or AUD, not both) fails at construction, not at request time | cfaccess | covered | `cfaccess/cfaccess_test.go::TestNewConfigModes` [u] |
| BR-SEC-006 | `/public/*` is never guarded — the shop front must work for anyone | cfaccess (path prefix) | partial | Only `/healthz` and `/items` are used as the non-admin examples; no test names a real `/public/*` path |
| BR-SEC-007 | Secrets (`MAIN_DB_URL`, `REDIS_URL`) never reach the startup config log | cmd `config` (`json:"-"`) | gap | Nothing asserts the dumped config omits them |
| BR-SEC-008 | An expired Access session re-authenticates instead of erroring: an `/admin/*` fetch answered with an Access redirect (`response.type === 'opaqueredirect'`, from `redirect: 'manual'`) reloads the page once per tab session so Cloudflare can serve its login page | PWA `src/api/client.ts` | gap | The frontend has no test tooling; verified by hand only |
| BR-SEC-009 | `/inventory*` navigations are never answered from the service-worker cache, so the Access login redirect always reaches the browser | `vite.config.ts` (`navigateFallbackDenylist`) | gap | The frontend has no test tooling; asserted by hand against the generated `dist/sw.js` |

### BR-OPS — boot, configuration and degraded modes

| ID | Rule | Enforced in | Status | Proof |
|---|---|---|---|---|
| BR-OPS-001 | `ENV` (development\|production) and `SERVER_PORT` are required; `MAIN_DB_URL` is required in production | cmd `config` | gap | `cmd/` has no test files at all |
| BR-OPS-002 | `S3_BASE_ENDPOINT` without `S3_UPLOAD_BUCKET` is a boot error | cmd `main.go` | gap | Same — the guard lives in `imageStore()` and is untested |
| BR-OPS-003 | With storage configured, boot pings (HeadBucket) the upload bucket and fails fast | cmd `main.go`, shared `aws_s3` | partial | The client's `Ping` is covered in the shared module (`environment/shared/golang/clients/aws_s3/client_test.go::TestPing`); main-api's boot wiring around it is not |
| BR-OPS-004 | Without a database the server still boots and serves only the platform routes | cmd `main.go`, `api/register.go` | gap | Nothing constructs the server without an application |
| BR-OPS-005 | Without object storage the photo routes are never registered, and `HasImageStore` is false — including on a nil `*Application` | app `app.go`, `api/register.go` | partial | `app/app_test.go::TestHasImageStore` + `TestNewFillsTheOptionalPorts` [u] prove the probe; **no test proves `NewServer` actually skips registration** |
| BR-OPS-006 | The builtin place seed runs at boot and fails the process, so a broken seed surfaces at deploy time | cmd `main.go` | partial | The seed operation is covered (BR-TAX-001); the boot-time fail-fast is not |
| BR-OPS-007 | Domain error kinds map to 400 / 404 / 409 / 413 / 503, and anything else is a 500 with no detail leaked | routes `errors.go`, domain | partial | Kind plumbing is covered (`domain/errors_test.go::TestKindOf`, `TestErrorIsMatchesOnKind` [u]; `app/items_test.go::TestRepositoryFailuresStayInternal` [u]) and 400/404/409 are asserted over HTTP in the integration suite. **413 and 503 are never asserted at the HTTP layer** |
| BR-OPS-008 | The client-visible error message is written in the domain; the HTTP layer only picks the status code | domain, routes | covered | `domain/errors_test.go::TestErrorMessage` [u]; `domain/item_test.go::TestParseStatus` [u] pins the exact wire strings |

### BR-FLOW — rules the PWA owns

Every rule in this area is a **gap**: `frontend/public_site` has no test runner,
no spec files, and no E2E suite. They are listed so the exposure is visible and
so an E2E pass has a target list — not to imply they are checked anywhere.

| ID | Rule | Enforced in |
|---|---|---|
| BR-FLOW-001 | The moves offered per status: draft → list/archive; listed → mark sold/unlist/archive; sold → none; archived → restore | `pages/Inventory.tsx` (`MOVES`), `pages/Item.tsx` |
| BR-FLOW-002 | Listing is blocked until the item has at least one selling place, with an explanation rather than a dead button | `pages/Item.tsx`, `pages/Inventory.tsx` |
| BR-FLOW-003 | Listing prompts for a list price greater than zero and sends it with the status change — a listed item is on the shop front immediately | `components/list-it-dialog.tsx` |
| BR-FLOW-004 | Mark-sold defaults to the listing price, today's date and the item's first selling place; "Custom…" creates a real place before recording the sale | `pages/Item.tsx` (`MarkSoldDialog`) |
| BR-FLOW-005 | Sold and archived items are records: nothing on them is editable | `pages/Item.tsx` (`readOnly`) |
| BR-FLOW-006 | Photos can be added only while the item is a draft | `pages/Item.tsx` (`canAddPhotos`) |
| BR-FLOW-007 | Delete is offered only for never-listed drafts and archived items, behind a confirm dialog | `pages/Item.tsx`, `pages/Inventory.tsx` |
| BR-FLOW-008 | Item details stay read-only until "Edit details"; cancel restores the loaded values | `pages/Item.tsx` (`editing`) |
| BR-FLOW-009 | Batch intake: saving clears the form, keeps places/labels ticked, counts the batch, and a failed photo upload never fails the saved item | `pages/Intake.tsx` |
| BR-FLOW-010 | Every photo is converted and resized client-side (≤1600 px, JPEG q0.82) before upload, so iOS HEIC never reaches the API | `components/image-upload.ts` |
| BR-FLOW-011 | The shop grid debounces search by 300 ms, restarts paging on any filter change, and shows "Show more" only when a cursor exists | `components/shop/catalog.tsx` |

---

## Gaps, ordered by what they cost

1. **BR-PHOTO-014 — the upload endpoint is untested.** `RegisterItemImageRoutes`
   is in no test. This is the one route that already broke in production (422,
   multipart spill to a `/tmp` that does not exist in a `FROM scratch` image),
   and nothing would catch it happening again. Highest value per test written.
2. **BR-FLOW-001…011 — no frontend tests exist.** Every flow rule that protects
   the owner from a bad move lives only in the PWA. Closing this needs tooling
   first (a runner for `frontend/public_site`), then the three highest-risk
   rules: listing needs a place and a price, sold/archived are read-only, delete
   is offered only where it is legal.
3. **BR-OPS-001…006 — boot and config are untested.** `cmd/` has no tests: a bad
   `ENV`, a storage endpoint with no bucket, or a no-database boot are all
   unproven. Cheap to cover — they are pure functions behind small seams.
4. **BR-ITEM-012 — admin list filters never run through the API against
   Postgres.** The integration test writes its own Ent queries instead of
   calling `GET /admin/items?place_id=…`, so the adapter's filter transcription
   is only proven against fakes.
5. **BR-TAX-003 / BR-OPS-007 — status-code mapping holes.** A duplicate name's
   409 and the 413/503 kinds are never asserted over HTTP.
6. **BR-SEC-007, BR-ITEM-001, BR-ITEM-004, BR-PHOTO-009** — small, individually
   cheap assertions that nothing currently makes.

## Known divergences

Documented decisions, not bugs to fix casually. Changing any of these is a
product decision first.

| What | Where | Why it stands |
|---|---|---|
| `StatusState` (update path) finds soft-deleted rows; `LiveStatusState` (delete path) does not | `ports/repositories.go`, ent adapter | The two handlers queried differently before the hexagonal refactor; preserved deliberately. Fake fidelity is locked in by `fakes/fakes_test.go::TestItemRepoSoftDeleteHidesFromReads` |
| The admin list loads photos per item; the catalog batches a page in one query | app `items.go` vs `catalog.go` | The public page can be large; the admin list is owner-only (BR-CAT-010) |
| The admin list reports `image_count` without storage; the public catalog reports `0` | app | Deliberate (BR-CAT-011) |
| Create with `status: sold` validates the place but never stores it | app `items.go` | Carried over from the old handler; likely a bug, left alone (BR-SALE-005) |

## Workflow

The pre-implementation discipline that `test-cases.md` used to carry now lives
here, rule-shaped instead of feature-shaped.

1. **Before implementing a feature**, add or extend the rules it introduces:
   one row per rule, `Status: planned`, with the level each will be proven at.
   A feature that adds no row is either pure plumbing or an unstated rule —
   decide which before writing the code.
2. **Before opening a PR**, audit the diff against those rows: flip `planned` to
   `covered`/`partial`, fill in the `Proof` cell with the real test function
   names, and add anything the plan missed. Scoped to the diff, not a repo sweep.
3. **When a rule changes**, update the row in the same change as the code — the
   registry is only useful while it is true. When a rule disappears, strike the
   row through with a date; never renumber or reuse an ID.
4. **When a test is renamed or deleted**, fix every `Proof` cell that names it.
   `grep -rn "func <Name>(" backend/main-api` is the check.
5. The coverage snapshot table is regenerated by hand when statuses move; it is
   a summary, not a source of truth.

## History

Condensed from `test-cases.md`, which this file replaces. Kept because it
records *when* each rule area got its coverage and why some of it looks the way
it does.

| Date order | Change | What it added |
|---|---|---|
| 1 | Cloudflare Access admin guard (`internal/cfaccess`) | The whole `BR-SEC` area, unit-tested with a local JWKS server and a generated key pair |
| 2 | Object storage boot wiring | Client behaviour covered in the shared `aws_s3` module; main-api's own wiring left uncovered — still true today (BR-OPS-003) |
| 3 | Inventory admin API (Phase 2 of `docs/plans/inventory-system.md`) | Items CRUD, search, mark-sold, places and labels; the first integration suite against local Postgres |
| 4 | Item status flow | `BR-STATUS`, `BR-DEL` and the sold-place rule, proven at the integration level first |
| 5 | Create responses carry a JSON body | Regression fix: bodyless huma responses 204'd and broke every inline "Add place/label" (BR-TAX-006) |
| 6 | Public catalog endpoints | The whole `BR-CAT` area, including the raw-JSON-key assertions that keep private fields off the wire |
| 7 | Ports-and-adapters refactor (`docs/plans/hexagonal-main-api.md`) | Added the unit level the `api` package never had: domain rules became pure functions testable without Postgres or S3. The 21 integration tests were the behaviour oracle and their assertions did not change |
| 8 | `service` merged into `app` | Structural only. Backfilled the untested public surface: `New` defaults, `HasImageStore` on a nil application, labels/places CRUD, measurement-unit parsing, `ListItemImages` without storage |
