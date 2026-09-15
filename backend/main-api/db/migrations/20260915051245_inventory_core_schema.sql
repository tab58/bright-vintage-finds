-- Create "selling_places" table
CREATE TABLE "public"."selling_places" (
  "id" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "is_builtin" boolean NOT NULL DEFAULT false,
  PRIMARY KEY ("id")
);
-- Create index "selling_places_name_key" to table: "selling_places"
CREATE UNIQUE INDEX "selling_places_name_key" ON "public"."selling_places" ("name");
-- Modify "items" table
ALTER TABLE "public"."items" ADD COLUMN "purchased_at" timestamptz NULL, ADD COLUMN "length" double precision NULL, ADD COLUMN "width" double precision NULL, ADD COLUMN "height" double precision NULL, ADD COLUMN "measurement_unit" character varying NOT NULL DEFAULT 'inch', ADD COLUMN "extra_measurements" text NULL, ADD COLUMN "weight_lbs" bigint NULL, ADD COLUMN "weight_oz" double precision NULL, ADD COLUMN "notes" text NULL, ADD COLUMN "whatnot_number" character varying NULL, ADD COLUMN "sold_place_id" character varying NULL, ADD CONSTRAINT "items_selling_places_sold_place" FOREIGN KEY ("sold_place_id") REFERENCES "public"."selling_places" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Create index "items_whatnot_number_key" to table: "items"
CREATE UNIQUE INDEX "items_whatnot_number_key" ON "public"."items" ("whatnot_number");
-- Create "labels" table
CREATE TABLE "public"."labels" (
  "id" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "labels_name_key" to table: "labels"
CREATE UNIQUE INDEX "labels_name_key" ON "public"."labels" ("name");
-- Create "item_labels" table
CREATE TABLE "public"."item_labels" (
  "item_id" character varying NOT NULL,
  "label_id" character varying NOT NULL,
  PRIMARY KEY ("item_id", "label_id"),
  CONSTRAINT "item_labels_item_id" FOREIGN KEY ("item_id") REFERENCES "public"."items" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "item_labels_label_id" FOREIGN KEY ("label_id") REFERENCES "public"."labels" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "item_selling_places" table
CREATE TABLE "public"."item_selling_places" (
  "item_id" character varying NOT NULL,
  "selling_place_id" character varying NOT NULL,
  PRIMARY KEY ("item_id", "selling_place_id"),
  CONSTRAINT "item_selling_places_item_id" FOREIGN KEY ("item_id") REFERENCES "public"."items" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "item_selling_places_selling_place_id" FOREIGN KEY ("selling_place_id") REFERENCES "public"."selling_places" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
