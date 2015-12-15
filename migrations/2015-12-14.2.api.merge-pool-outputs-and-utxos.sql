ALTER TABLE utxos ADD confirmed BOOLEAN NOT NULL;
ALTER TABLE utxos ADD pool_tx_hash TEXT REFERENCES pool_txs (tx_hash);
ALTER TABLE utxos RENAME COLUMN txid TO tx_hash;
ALTER TABLE utxos ADD CHECK (confirmed = (pool_tx_hash IS NULL));
ALTER TABLE utxos ADD CHECK (confirmed = (block_hash IS NOT NULL));
ALTER TABLE utxos ADD CHECK (confirmed = (block_height IS NOT NULL));
INSERT INTO utxos (tx_hash, pool_tx_hash, index, asset_id, amount,
                   addr_index, account_id, contract_hash,
                   manager_node_id, reserved_until,
                   metadata, script, confirmed)
    SELECT tx_hash, tx_hash, index, asset_id, amount,
           addr_index, account_id, contract_hash,
           manager_node_id, reserved_until,
           metadata, script, FALSE
        FROM pool_outputs;
DROP TABLE pool_outputs;
