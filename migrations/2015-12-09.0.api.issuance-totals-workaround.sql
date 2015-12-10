CREATE TABLE issuance_totals (
	asset_id text PRIMARY KEY,
	pool bigint DEFAULT 0 NOT NULL CHECK (pool >= 0),
	confirmed bigint DEFAULT 0 NOT NULL CHECK (confirmed >= 0)
);

INSERT INTO issuance_totals
SELECT id, issued_pool, issued_confirmed
FROM assets;

ALTER TABLE assets
	DROP COLUMN issued_pool,
	DROP COLUMN issued_confirmed;
