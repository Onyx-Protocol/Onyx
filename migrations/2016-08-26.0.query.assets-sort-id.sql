ALTER TABLE annotated_assets ADD COLUMN sort_id text NOT NULL;
CREATE INDEX annotated_assets_sort_id ON annotated_assets USING btree (sort_id);
