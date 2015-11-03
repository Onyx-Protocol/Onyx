ALTER TABLE utxos ADD COLUMN script bytea DEFAULT '\x'::bytea NOT NULL;
