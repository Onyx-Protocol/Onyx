ALTER TABLE assets ADD COLUMN inner_asset_id text;
ALTER TABLE assets ADD COLUMN issuance_script bytea NOT NULL;

ALTER TABLE assets DROP COLUMN definition_url; -- because it can be stored in definition

CREATE TABLE asset_definition_pointers (
  asset_id text PRIMARY KEY,
  asset_definition_hash text NOT NULL,
  transaction_hash text NOT NULL,
  input_index bigint NOT NULL
);

CREATE TABLE asset_definitions (
  hash text PRIMARY KEY,
  definition bytea
);