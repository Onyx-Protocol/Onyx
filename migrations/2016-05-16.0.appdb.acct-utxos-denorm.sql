ALTER TABLE account_utxos
	ADD COLUMN script bytea,
	ADD COLUMN metadata bytea;

UPDATE account_utxos AS a
SET script=u.script, metadata=u.metadata
FROM utxos u
WHERE ((a.tx_hash, a.index) = (u.tx_hash, u.index));

ALTER TABLE account_utxos
	ALTER COLUMN script SET NOT NULL,
	ALTER COLUMN metadata SET NOT NULL;
