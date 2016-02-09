
CREATE TABLE account_utxos (
    tx_hash TEXT NOT NULL,
    index INTEGER NOT NULL,
    asset_id TEXT NOT NULL,
    amount bigint NOT NULL,
    manager_node_id text NOT NULL,
    account_id TEXT NOT NULL,
    addr_index bigint NOT NULL,
    reserved_until timestamptz NOT NULL DEFAULT '1970-01-01',
    PRIMARY KEY (tx_hash, index),
    CONSTRAINT account_utxos_fkey FOREIGN KEY (tx_hash, index) REFERENCES utxos ON DELETE CASCADE
);

CREATE INDEX account_utxos_account_id ON account_utxos (account_id);

CREATE TABLE blocks_utxos (
    tx_hash TEXT NOT NULL,
    index INTEGER NOT NULL,
    PRIMARY KEY (tx_hash, index),
    FOREIGN KEY (tx_hash, index) REFERENCES utxos ON DELETE CASCADE
);

INSERT INTO blocks_utxos
SELECT tx_hash, index
FROM utxos WHERE pool_tx_hash IS NULL;

INSERT INTO account_utxos
SELECT tx_hash, index, asset_id, amount, manager_node_id, account_id, addr_index, reserved_until FROM utxos;

ALTER TABLE utxos
	DROP COLUMN addr_index,
	DROP COLUMN account_id,
	DROP COLUMN manager_node_id,
	DROP COLUMN confirmed,
	DROP COLUMN pool_tx_hash,
	DROP COLUMN block_hash,
	DROP COLUMN block_height,
  DROP COLUMN reserved_until;

CREATE VIEW utxos_status AS
SELECT u.*, b.tx_hash IS NOT NULL AS confirmed
FROM utxos u
LEFT JOIN blocks_utxos b ON (u.tx_hash, u.index) = (b.tx_hash, b.index);

CREATE INDEX ON account_utxos (account_id, asset_id);
CREATE INDEX ON account_utxos (manager_node_id, asset_id);
