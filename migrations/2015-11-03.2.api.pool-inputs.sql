ALTER TABLE utxos ADD COLUMN metadata bytea NOT NULL DEFAULT '';
ALTER TABLE pool_outputs ADD COLUMN metadata bytea NOT NULL DEFAULT '';

CREATE TABLE pool_inputs (
	tx_hash text,
	index integer,
	PRIMARY KEY (tx_hash, index)
)
