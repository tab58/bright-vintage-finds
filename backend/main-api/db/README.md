# db (package db_platform)

Postgres persistence layer. **Ent** ORM + **Atlas** migrations.

## Folder Structure

```
db/
├── schema/           # Ent schema definitions (source of truth for DB structure)
│   └── mixin/        # Reusable field sets applied across schemas
├── generated/        # Ent-generated Go code (NEVER hand-edit)
├── migrations/       # Atlas-managed SQL migration files (NEVER hand-edit)
├── client.go         # Postgres client wrapper (pgx driver)
└── generate.go       # go:generate directive for Ent codegen
```

## Mixins

| Mixin | Fields | Purpose |
|---|---|---|
| `WithSortableID` | `id` (KSUID, immutable) | Sortable IDs: creation order = alphabetical order |
| `WithCreationTracking` | `created_at` | Immutable creation timestamp |
| `WithUpdateTracking` | `updated_at`, `version` | Last modification; `version` for optimistic locking |
| `WithSoftDelete` | `deleted_at` (nullable) | Soft deletion — filter with `DeletedAtIsNil()` |

## Entities

```
erDiagram
    USER ||--o{ ITEM : "owns"
    ITEM ||--o{ ITEM_IMAGE : "has"

    USER {
        string idp_id UK "immutable"
        string email  UK
        string full_name
        enum   account_status "active|inactive|suspended"
    }

    ITEM {
        string name
        string description            "nullable"
        string category               "nullable"
        string condition              "nullable"
        enum   status                 "draft|listed|sold|archived"
        int64  acquisition_cost_cents "nullable"
        int64  listing_price_cents    "nullable"
        int64  sold_price_cents       "nullable"
        timestamp sold_at             "nullable"
        string user_items FK          "owner"
    }

    ITEM_IMAGE {
        string upload_bucket "immutable"
        string upload_key    "immutable"
        string filename      "nullable"
        string content_type  "nullable"
        int64  size_bytes    "nullable"
        int    display_order "default 0"
        string item_images FK "immutable"
    }
```

Prices are USD cents. Sales insight is computed from `sold_price_cents`/`sold_at` on sold items — no separate sales table yet.

## Workflow: Adding or Modifying a Schema

1. Edit or create the schema file in `schema/`
2. Run Ent codegen: `go generate ./db/...`
3. Generate the Atlas migration: `task generate-migration -- <descriptive_name>` (snake_case name; requires Docker)
4. Review the generated SQL in `migrations/`
5. Apply to local DB: `task apply-migrations`

## Rules

- Never hand-edit `migrations/*.sql` or `atlas.sum` — generate a corrective migration instead.
- Always run codegen before generating a migration (Atlas reads generated Ent code).
- Never hand-edit `generated/`.
