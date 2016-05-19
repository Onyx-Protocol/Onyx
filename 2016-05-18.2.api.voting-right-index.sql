ALTER TABLE voting_right_txs
  DROP CONSTRAINT voting_right_txs_pkey,
  DROP COLUMN block_tx_index,
  DROP COLUMN void,
  ADD COLUMN void_block_height int NULL,
  ADD COLUMN ordinal int NOT NULL,
  ADD PRIMARY KEY (asset_id, ordinal);
ALTER TABLE voting_right_txs
  RENAME TO voting_rights;
ALTER TABLE voting_tokens
  ADD COLUMN block_height int NOT NULL;
