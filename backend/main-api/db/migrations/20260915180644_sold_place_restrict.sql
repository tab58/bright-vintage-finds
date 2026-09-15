-- Modify "items" table
ALTER TABLE "public"."items" DROP CONSTRAINT "items_selling_places_sold_place", ADD CONSTRAINT "items_selling_places_sold_place" FOREIGN KEY ("sold_place_id") REFERENCES "public"."selling_places" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
