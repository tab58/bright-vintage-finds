-- Create "users" table
CREATE TABLE "public"."users" (
  "id" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "deleted_at" timestamptz NULL,
  "idp_id" character varying NOT NULL,
  "email" character varying NOT NULL,
  "full_name" character varying NOT NULL,
  "account_status" character varying NOT NULL DEFAULT 'inactive',
  PRIMARY KEY ("id")
);
-- Create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX "users_email_key" ON "public"."users" ("email");
-- Create index "users_idp_id_key" to table: "users"
CREATE UNIQUE INDEX "users_idp_id_key" ON "public"."users" ("idp_id");
-- Create "items" table
CREATE TABLE "public"."items" (
  "id" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "category" character varying NULL,
  "condition" character varying NULL,
  "status" character varying NOT NULL DEFAULT 'draft',
  "acquisition_cost_cents" bigint NULL,
  "listing_price_cents" bigint NULL,
  "sold_price_cents" bigint NULL,
  "sold_at" timestamptz NULL,
  "user_items" character varying NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "items_users_items" FOREIGN KEY ("user_items") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "item_images" table
CREATE TABLE "public"."item_images" (
  "id" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "upload_bucket" character varying NOT NULL,
  "upload_key" character varying NOT NULL,
  "filename" character varying NULL,
  "content_type" character varying NULL,
  "size_bytes" bigint NULL,
  "display_order" bigint NOT NULL DEFAULT 0,
  "item_images" character varying NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "item_images_items_images" FOREIGN KEY ("item_images") REFERENCES "public"."items" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
