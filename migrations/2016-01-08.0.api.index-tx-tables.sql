-- Primary keys

CREATE UNIQUE INDEX CONCURRENTLY manager_txs_id_idx ON manager_txs (id);
ALTER TABLE manager_txs ADD CONSTRAINT manager_txs_add_pkey PRIMARY KEY USING INDEX manager_txs_id_idx;

CREATE UNIQUE INDEX CONCURRENTLY issuer_txs_id_idx ON issuer_txs (id);
ALTER TABLE issuer_txs ADD CONSTRAINT issuer_txs_add_pkey PRIMARY KEY USING INDEX issuer_txs_id_idx;

-- Index by node, pre-sorted by ID to speed up retrieval

CREATE INDEX CONCURRENTLY ON manager_txs (manager_node_id, id DESC);
CREATE INDEX CONCURRENTLY ON issuer_txs (issuer_node_id, id DESC);

-- Junction tables
-- Lead with the account or asset ID, since those will be used as the primary filter.

CREATE UNIQUE INDEX CONCURRENTLY ON manager_txs_accounts (account_id, manager_tx_id DESC);
CREATE UNIQUE INDEX CONCURRENTLY ON issuer_txs_assets (asset_id, issuer_tx_id DESC);
