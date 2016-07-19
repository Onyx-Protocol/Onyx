CREATE UNIQUE INDEX manager_txs_node_hash
ON manager_txs (manager_node_id, tx_hash);
CREATE UNIQUE INDEX issuer_txs_node_hash
ON issuer_txs (issuer_node_id, tx_hash);
