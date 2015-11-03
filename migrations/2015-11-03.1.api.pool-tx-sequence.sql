CREATE SEQUENCE pool_tx_sort_id_seq;
ALTER TABLE pool_txs ADD COLUMN sort_id text DEFAULT nextval('pool_tx_sort_id_seq') UNIQUE NOT NULL;
