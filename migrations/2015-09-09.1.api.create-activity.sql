CREATE TABLE activity (
    id text DEFAULT next_chain_id('act'::text) NOT NULL PRIMARY KEY,
    wallet_id text NOT NULL REFERENCES wallets (id),
    data json NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    txid text NOT NULL
);

CREATE TABLE activity_buckets (
    activity_id text NOT NULL REFERENCES activity (id),
    bucket_id text NOT NULL REFERENCES buckets (id)
);

CREATE UNIQUE INDEX ON activity USING btree (wallet_id, txid);
CREATE INDEX ON activity (wallet_id);

CREATE UNIQUE INDEX ON activity_buckets USING btree (activity_id, bucket_id);
CREATE INDEX ON activity_buckets (bucket_id);