CREATE SEQUENCE chain_id_seq;

CREATE FUNCTION next_chain_id(prefix text) RETURNS text
    LANGUAGE plpgsql
    AS $$
DECLARE
	our_epoch_ms bigint := 1433333333333; -- do not change
	seq_id bigint;
	now_ms bigint;     -- from unix epoch, not ours
	shard_id int := 4; -- must be different on each shard
	n bigint;
BEGIN
	SELECT nextval('chain_id_seq') % 1024 INTO seq_id;
	SELECT FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000) INTO now_ms;
	n := (now_ms - our_epoch_ms) << 23;
	n := n | (shard_id << 10);
	n := n | (seq_id);
	RETURN prefix || b32enc_crockford(int8send(n));
END;
$$;

CREATE TABLE wallets (
    id text DEFAULT next_chain_id('w'::text) NOT NULL PRIMARY KEY,
    application_id text NOT NULL,
    development boolean,
    block_chain text,
    sigs_required integer DEFAULT 2 NOT NULL,
    chain_keys integer DEFAULT 1 NOT NULL,
    key_index bigserial not null,
    label text NOT NULL,
    current_rotation text NOT NULL,
    pek text NOT NULL,
    next_asset_index bigint DEFAULT 0 NOT NULL,
    next_bucket_index bigint DEFAULT 0 NOT NULL,
    buckets_count bigint DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX ON wallets USING btree (application_id);

CREATE TABLE assets (
    id text NOT NULL PRIMARY KEY,
    wallet_id text NOT NULL REFERENCES wallets (id),
    key_index bigint NOT NULL,
    keys text[] DEFAULT '{}'::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    definition_mutable boolean DEFAULT false NOT NULL,
    definition_url text DEFAULT ''::text NOT NULL,
    definition bytea,
    redeem_script bytea
);

CREATE TABLE buckets (
    id text DEFAULT next_chain_id('b'::text) PRIMARY KEY,
    wallet_id text NOT NULL REFERENCES wallets (id),
    key_index bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    next_receiver_index bigint DEFAULT 0 NOT NULL
);

CREATE UNIQUE INDEX buckets_wallet_path ON buckets USING btree (wallet_id, key_index);

CREATE TABLE keys (
		id text NOT NULL PRIMARY KEY,
    type text,
    xpub text,
    enc_xpriv text,
    created_at timestamp with time zone DEFAULT now()
);

CREATE TABLE receivers (
    id text DEFAULT next_chain_id('r'::text) NOT NULL,
    wallet_id text NOT NULL REFERENCES wallets (id),
    bucket_id text NOT NULL REFERENCES buckets (id),
    keyset text[] NOT NULL,
    key_index bigint NOT NULL,
    address text NOT NULL UNIQUE,
    memo text,
    amount bigint,
    is_change boolean DEFAULT false NOT NULL,
    expiration timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX ON receivers USING btree (bucket_id, key_index);
CREATE INDEX ON receivers USING btree (bucket_id);
CREATE INDEX ON receivers USING btree (wallet_id);

CREATE TABLE rotations (
    id text DEFAULT next_chain_id('rot'::text) NOT NULL PRIMARY KEY,
    wallet_id text NOT NULL REFERENCES wallets (id),
    pek_pub text,
    keyset text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
