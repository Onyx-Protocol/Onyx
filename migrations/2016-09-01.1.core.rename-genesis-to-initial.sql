ALTER TABLE config RENAME COLUMN genesis_hash TO initial_block_hash;
ALTER TABLE assets RENAME COLUMN genesis_hash TO initial_block_hash;
