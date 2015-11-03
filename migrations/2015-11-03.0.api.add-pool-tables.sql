CREATE TABLE pool_txs (
	tx_hash text PRIMARY KEY,
	data bytea NOT NULL
);

CREATE TABLE pool_outputs (
	tx_hash text NOT NULL REFERENCES pool_txs ON DELETE CASCADE,
	index integer NOT NULL,
	asset_id text NOT NULL,
	issuance_id text,
	script bytea NOT NULL,
	amount bigint NOT NULL,
	spent bool DEFAULT false NOT NULL,

	-- Manager info; to be factored out along with
	-- the corresponding columns in table utxos.
	addr_index bigint NOT NULL,
	account_id text NOT NULL,
	manager_node_id text NOT NULL,
	reserved_until timestamp with time zone DEFAULT '1979-12-31 16:00:00-08'::timestamp with time zone NOT NULL,

	PRIMARY KEY (tx_hash, index)
);

ALTER TABLE utxos
	ADD COLUMN block_hash text,     -- make NOT NULL once we start using pool_outputs
	ADD COLUMN block_height bigint; -- make NOT NULL once we start using pool_outputs
