-- We'll stage this migration into two (really three) steps:
-- 1. Add the column and begin to populate it.
-- 2a. Backfill the columns for existing data.
-- 2b. Add constraints and indexes.

ALTER TABLE blocks_txs
	ADD COLUMN block_height bigint, -- TODO: NOT NULL
	ADD COLUMN block_pos int; -- TODO: NOT NULL

-- TODO: unique index on (block_height, tx_block_pos)
