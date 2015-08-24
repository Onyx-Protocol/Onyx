CREATE TABLE outputs (
	-- block chain tx data
	txid text NOT NULL,
	index integer NOT NULL,
	asset_id text NOT NULL,
	amount bigint NOT NULL,

	-- chain bookkeeping
	receiver_id text NOT NULL,
	bucket_id text NOT NULL,
	wallet_id text NOT NULL,
	reserved_at timestamptz NOT NULL DEFAULT '1980-01-01',
	created_at timestamptz NOT NULL DEFAULT now(),

	PRIMARY KEY (txid, index)
);

CREATE INDEX ON outputs (receiver_id, asset_id, reserved_at);
CREATE INDEX ON outputs (bucket_id, asset_id, reserved_at);
CREATE INDEX ON outputs (wallet_id, asset_id, reserved_at);

CREATE EXTENSION plv8;
