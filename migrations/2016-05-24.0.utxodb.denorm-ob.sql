ALTER TABLE orderbook_utxos
	ADD COLUMN asset_id text,
	ADD COLUMN amount bigint,
	ADD COLUMN script bytea;

UPDATE orderbook_utxos AS o
SET asset_id=u.asset_id, amount=u.amount, script=u.script
FROM utxos u
WHERE ((o.tx_hash, o.index) = (u.tx_hash, u.index));

ALTER TABLE orderbook_utxos
	ALTER COLUMN asset_id SET NOT NULL,
	ALTER COLUMN amount SET NOT NULL,
	ALTER COLUMN script SET NOT NULL;
