CREATE TABLE asset_tags (
  asset_id text NOT NULL,
  tags jsonb,
  UNIQUE(asset_id)
);

