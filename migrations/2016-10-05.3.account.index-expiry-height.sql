CREATE INDEX ON account_utxos (expiry_height) WHERE confirmed_in IS NULL;
