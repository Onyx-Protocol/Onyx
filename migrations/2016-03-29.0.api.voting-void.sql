ALTER TABLE voting_right_txs
    ADD COLUMN block_height BIGINT NOT NULL,
    ADD COLUMN block_tx_index INTEGER NOT NULL,
    ADD COLUMN void BOOLEAN NOT NULL DEFAULT false;
