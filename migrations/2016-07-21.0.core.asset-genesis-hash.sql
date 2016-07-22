ALTER TABLE assets ADD COLUMN genesis_hash text NOT NULL;
UPDATE assets SET genesis_hash = (SELECT block_hash FROM blocks WHERE height = 1);
