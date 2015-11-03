CREATE TABLE txs (
	tx_hash text NOT NULL PRIMARY KEY,
	data bytea NOT NULL
);
CREATE TABLE blocks_txs (
	tx_hash text NOT NULL,
	block_hash text NOT NULL,
	UNIQUE(tx_hash, block_hash)
);
