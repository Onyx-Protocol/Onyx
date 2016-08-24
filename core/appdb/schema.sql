--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.0
-- Dumped by pg_dump version 9.5.0

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
--



SET search_path = public, pg_catalog;

--
-- Name: b32enc_crockford(bytea); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION b32enc_crockford(src bytea) RETURNS text
    LANGUAGE plpgsql IMMUTABLE
    AS $$
	-- Adapted from the Go package encoding/base32.
	-- See https://golang.org/src/encoding/base32/base32.go.
	-- NOTE(kr): this function does not pad its output
DECLARE
	-- alphabet is the base32 alphabet defined
	-- by Douglas Crockford. It preserves lexical
	-- order and avoids visually-similar symbols.
	-- See http://www.crockford.com/wrmg/base32.html.
	alphabet text := '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
	dst text := '';
	n integer;
	b0 integer;
	b1 integer;
	b2 integer;
	b3 integer;
	b4 integer;
	b5 integer;
	b6 integer;
	b7 integer;
BEGIN
	FOR r IN 0..(length(src)-1) BY 5
	LOOP
		b0:=0; b1:=0; b2:=0; b3:=0; b4:=0; b5:=0; b6:=0; b7:=0;

		-- Unpack 8x 5-bit source blocks into an 8 byte
		-- destination quantum
		n := length(src) - r;
		IF n >= 5 THEN
			b7 := get_byte(src, r+4) & 31;
			b6 := get_byte(src, r+4) >> 5;
		END IF;
		IF n >= 4 THEN
			b6 := b6 | (get_byte(src, r+3) << 3) & 31;
			b5 := (get_byte(src, r+3) >> 2) & 31;
			b4 := get_byte(src, r+3) >> 7;
		END IF;
		IF n >= 3 THEN
			b4 := b4 | (get_byte(src, r+2) << 1) & 31;
			b3 := (get_byte(src, r+2) >> 4) & 31;
		END IF;
		IF n >= 2 THEN
			b3 := b3 | (get_byte(src, r+1) << 4) & 31;
			b2 := (get_byte(src, r+1) >> 1) & 31;
			b1 := (get_byte(src, r+1) >> 6) & 31;
		END IF;
		b1 := b1 | (get_byte(src, r) << 2) & 31;
		b0 := get_byte(src, r) >> 3;

		-- Encode 5-bit blocks using the base32 alphabet
		dst := dst || substr(alphabet, b0+1, 1);
		dst := dst || substr(alphabet, b1+1, 1);
		IF n >= 2 THEN
			dst := dst || substr(alphabet, b2+1, 1);
			dst := dst || substr(alphabet, b3+1, 1);
		END IF;
		IF n >= 3 THEN
			dst := dst || substr(alphabet, b4+1, 1);
		END IF;
		IF n >= 4 THEN
			dst := dst || substr(alphabet, b5+1, 1);
			dst := dst || substr(alphabet, b6+1, 1);
		END IF;
		IF n >= 5 THEN
			dst := dst || substr(alphabet, b7+1, 1);
		END IF;
	END LOOP;
	RETURN dst;
END;
$$;


--
-- Name: cancel_reservation(integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION cancel_reservation(inp_reservation_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM reservations WHERE reservation_id = inp_reservation_id;
END;
$$;


--
-- Name: create_reservation(text, text, timestamp with time zone, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION create_reservation(inp_asset_id text, inp_account_id text, inp_expiry timestamp with time zone, inp_idempotency_key text, OUT reservation_id integer, OUT already_existed boolean, OUT existing_change bigint) RETURNS record
    LANGUAGE plpgsql
    AS $$
DECLARE
    row RECORD;
BEGIN
    INSERT INTO reservations (asset_id, account_id, expiry, idempotency_key)
        VALUES (inp_asset_id, inp_account_id, inp_expiry, inp_idempotency_key)
        ON CONFLICT (idempotency_key) DO NOTHING
        RETURNING reservations.reservation_id, FALSE AS already_existed, CAST(0 AS BIGINT) AS existing_change INTO row;
    -- Iff the insert was successful, then a row is returned. The IF NOT FOUND check
    -- will be true iff the insert failed because the row already exists.
    IF NOT FOUND THEN
        SELECT r.reservation_id, TRUE AS already_existed, r.change AS existing_change INTO STRICT row
            FROM reservations r
            WHERE r.idempotency_key = inp_idempotency_key;
    END IF;
    reservation_id := row.reservation_id;
    already_existed := row.already_existed;
    existing_change := row.existing_change;
END;
$$;


--
-- Name: expire_reservations(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION expire_reservations() RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM reservations WHERE expiry < CURRENT_TIMESTAMP;
END;
$$;


--
-- Name: key_index(bigint); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION key_index(n bigint) RETURNS integer[]
    LANGUAGE plpgsql
    AS $$
DECLARE
	maxint32 int := x'7fffffff'::int;
BEGIN
	RETURN ARRAY[(n>>31) & maxint32, n & maxint32];
END;
$$;


--
-- Name: next_chain_id(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION next_chain_id(prefix text) RETURNS text
    LANGUAGE plpgsql
    AS $$
	-- Adapted from the technique published by Instagram.
	-- See http://instagram-engineering.tumblr.com/post/10853187575/sharding-ids-at-instagram.
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


--
-- Name: reserve_utxo(text, bigint, timestamp with time zone, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION reserve_utxo(inp_tx_hash text, inp_out_index bigint, inp_expiry timestamp with time zone, inp_idempotency_key text) RETURNS record
    LANGUAGE plpgsql
    AS $$
DECLARE
    res RECORD;
    row RECORD;
    ret RECORD;
BEGIN
    SELECT * FROM create_reservation(NULL, NULL, inp_expiry, inp_idempotency_key) INTO STRICT res;
    IF res.already_existed THEN
      SELECT res.reservation_id, res.already_existed, res.existing_change, CAST(0 AS BIGINT) AS amount, FALSE AS insufficient INTO ret;
      RETURN ret;
    END IF;

    SELECT tx_hash, index, amount INTO row
        FROM account_utxos u
        WHERE inp_tx_hash = tx_hash
              AND inp_out_index = index
              AND reservation_id IS NULL
        LIMIT 1
        FOR UPDATE
        SKIP LOCKED;
    IF FOUND THEN
        UPDATE account_utxos SET reservation_id = res.reservation_id
            WHERE (tx_hash, index) = (row.tx_hash, row.index);
    ELSE
      PERFORM cancel_reservation(res.reservation_id);
      res.reservation_id := 0;
    END IF;

    SELECT res.reservation_id, res.already_existed, EXISTS(SELECT tx_hash FROM account_utxos WHERE tx_hash = inp_tx_hash AND index = inp_out_index) INTO ret;
    RETURN ret;
END;
$$;


--
-- Name: reserve_utxos(text, text, text, bigint, bigint, timestamp with time zone, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION reserve_utxos(inp_asset_id text, inp_account_id text, inp_tx_hash text, inp_out_index bigint, inp_amt bigint, inp_expiry timestamp with time zone, inp_idempotency_key text) RETURNS record
    LANGUAGE plpgsql
    AS $$
DECLARE
    res RECORD;
    row RECORD;
    ret RECORD;
    available BIGINT := 0;
    unavailable BIGINT := 0;
BEGIN
    SELECT * FROM create_reservation(inp_asset_id, inp_account_id, inp_expiry, inp_idempotency_key) INTO STRICT res;
    IF res.already_existed THEN
      SELECT res.reservation_id, res.already_existed, res.existing_change, CAST(0 AS BIGINT) AS amount, FALSE AS insufficient INTO ret;
      RETURN ret;
    END IF;

    LOOP
        SELECT tx_hash, index, amount INTO row
            FROM account_utxos u
            WHERE asset_id = inp_asset_id
                  AND inp_account_id = account_id
                  AND (inp_tx_hash IS NULL OR inp_tx_hash = tx_hash)
                  AND (inp_out_index IS NULL OR inp_out_index = index)
                  AND reservation_id IS NULL
            LIMIT 1
            FOR UPDATE
            SKIP LOCKED;
        IF FOUND THEN
            UPDATE account_utxos SET reservation_id = res.reservation_id
                WHERE (tx_hash, index) = (row.tx_hash, row.index);
            available := available + row.amount;
            IF available >= inp_amt THEN
                EXIT;
            END IF;
        ELSE
            EXIT;
        END IF;
    END LOOP;

    IF available < inp_amt THEN
        SELECT SUM(change) AS change INTO STRICT row
            FROM reservations
            WHERE asset_id = inp_asset_id AND account_id = inp_account_id;
        unavailable := row.change;
        PERFORM cancel_reservation(res.reservation_id);
        res.reservation_id := 0;
    ELSE
        UPDATE reservations SET change = available - inp_amt
            WHERE reservation_id = res.reservation_id;
    END IF;

    SELECT res.reservation_id, res.already_existed, CAST(0 AS BIGINT) AS existing_change, available AS amount, (available+unavailable < inp_amt) AS insufficient INTO ret;
    RETURN ret;
END;
$$;


--
-- Name: to_key_index(integer[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION to_key_index(n integer[]) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
BEGIN
	RETURN n[1]::bigint<<31 | n[2]::bigint;
END;
$$;


--
-- Name: account_control_program_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE account_control_program_seq
    START WITH 10001
    INCREMENT BY 10000
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: account_control_programs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_control_programs (
    id text DEFAULT next_chain_id('acp'::text) NOT NULL,
    signer_id text NOT NULL,
    key_index bigint NOT NULL,
    control_program bytea NOT NULL,
    redeem_program bytea NOT NULL
);


--
-- Name: account_utxos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_utxos (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    amount bigint NOT NULL,
    account_id text NOT NULL,
    control_program_index bigint NOT NULL,
    reservation_id integer,
    control_program bytea NOT NULL,
    metadata bytea NOT NULL,
    confirmed_in bigint,
    block_pos integer,
    block_timestamp bigint,
    expiry_height bigint
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE accounts (
    account_id text NOT NULL,
    tags jsonb,
    alias text,
    archived boolean DEFAULT false NOT NULL
);


--
-- Name: annotated_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_accounts (
    id text NOT NULL,
    data jsonb NOT NULL
);


--
-- Name: annotated_assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_assets (
    id text NOT NULL,
    data jsonb NOT NULL
);


--
-- Name: annotated_outputs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_outputs (
    block_height bigint NOT NULL,
    tx_pos integer NOT NULL,
    output_index integer NOT NULL,
    tx_hash text NOT NULL,
    data jsonb NOT NULL,
    timespan int8range NOT NULL
);


--
-- Name: annotated_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_txs (
    block_height bigint NOT NULL,
    tx_pos integer NOT NULL,
    tx_hash text NOT NULL,
    data jsonb NOT NULL
);


--
-- Name: asset_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE asset_tags (
    asset_id text NOT NULL,
    tags jsonb
);


--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE assets (
    id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    definition_mutable boolean DEFAULT false NOT NULL,
    sort_id text DEFAULT next_chain_id('asset'::text) NOT NULL,
    issuance_program bytea NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    client_token text,
    genesis_hash text NOT NULL,
    signer_id text NOT NULL,
    definition jsonb,
    alias text
);


--
-- Name: assets_key_index_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE assets_key_index_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE blocks (
    block_hash text NOT NULL,
    height bigint NOT NULL,
    data bytea NOT NULL,
    header bytea NOT NULL
);


--
-- Name: blocks_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE blocks_txs (
    tx_hash text NOT NULL,
    block_hash text NOT NULL,
    block_height bigint NOT NULL,
    block_pos integer NOT NULL
);


--
-- Name: chain_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE chain_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: generator_pending_block; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE generator_pending_block (
    singleton boolean DEFAULT true NOT NULL,
    data bytea NOT NULL,
    CONSTRAINT generator_pending_block_singleton CHECK (singleton)
);


--
-- Name: issuance_totals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE issuance_totals (
    asset_id text NOT NULL,
    issued bigint DEFAULT 0 NOT NULL,
    destroyed bigint DEFAULT 0 NOT NULL,
    height bigint NOT NULL,
    CONSTRAINT issuance_totals_confirmed_check CHECK ((issued >= 0)),
    CONSTRAINT positive_destroyed_confirmed CHECK ((destroyed >= 0))
);


--
-- Name: leader; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE leader (
    singleton boolean DEFAULT true NOT NULL,
    leader_key text NOT NULL,
    expiry timestamp with time zone DEFAULT '1970-01-01 00:00:00-08'::timestamp with time zone NOT NULL,
    CONSTRAINT leader_singleton CHECK (singleton)
);


--
-- Name: migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE migrations (
    filename text NOT NULL,
    hash text NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: mockhsm; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE mockhsm (
    xpub bytea NOT NULL,
    xprv bytea NOT NULL,
    xpub_hash text NOT NULL,
    alias text
);


--
-- Name: pool_tx_sort_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE pool_tx_sort_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pool_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE pool_txs (
    tx_hash text NOT NULL,
    data bytea NOT NULL,
    sort_id bigint DEFAULT nextval('pool_tx_sort_id_seq'::regclass) NOT NULL
);


--
-- Name: query_blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE query_blocks (
    height bigint NOT NULL,
    "timestamp" bigint NOT NULL
);


--
-- Name: query_indexes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE query_indexes (
    id text DEFAULT next_chain_id('idx'::text) NOT NULL,
    alias text NOT NULL,
    type text NOT NULL,
    query text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sum_by text[]
);


--
-- Name: query_indexes_internal_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE query_indexes_internal_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: query_indexes_internal_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE query_indexes_internal_id_seq OWNED BY query_indexes.id;


--
-- Name: reservation_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE reservation_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: reservations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE reservations (
    reservation_id integer DEFAULT nextval('reservation_seq'::regclass) NOT NULL,
    asset_id text,
    account_id text,
    expiry timestamp with time zone DEFAULT '1970-01-01 00:00:00-08'::timestamp with time zone NOT NULL,
    change bigint DEFAULT 0 NOT NULL,
    idempotency_key text
);


--
-- Name: signed_blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE signed_blocks (
    block_height bigint NOT NULL,
    block_hash text NOT NULL
);


--
-- Name: signers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE signers (
    id text NOT NULL,
    type text NOT NULL,
    key_index bigint NOT NULL,
    xpubs text[] NOT NULL,
    quorum integer NOT NULL,
    client_token text,
    archived boolean DEFAULT false NOT NULL
);


--
-- Name: signers_key_index_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE signers_key_index_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: signers_key_index_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE signers_key_index_seq OWNED BY signers.key_index;


--
-- Name: snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE snapshots (
    height bigint NOT NULL,
    data bytea NOT NULL
);


--
-- Name: txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE txs (
    tx_hash text NOT NULL,
    data bytea NOT NULL
);


--
-- Name: key_index; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY signers ALTER COLUMN key_index SET DEFAULT nextval('signers_key_index_seq'::regclass);


--
-- Name: account_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT account_tags_pkey PRIMARY KEY (account_id);


--
-- Name: account_utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_pkey PRIMARY KEY (tx_hash, index);


--
-- Name: accounts_alias_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT accounts_alias_key UNIQUE (alias);


--
-- Name: annotated_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY annotated_accounts
    ADD CONSTRAINT annotated_accounts_pkey PRIMARY KEY (id);


--
-- Name: annotated_assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY annotated_assets
    ADD CONSTRAINT annotated_assets_pkey PRIMARY KEY (id);


--
-- Name: annotated_outputs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY annotated_outputs
    ADD CONSTRAINT annotated_outputs_pkey PRIMARY KEY (block_height, tx_pos, output_index);


--
-- Name: annotated_txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY annotated_txs
    ADD CONSTRAINT annotated_txs_pkey PRIMARY KEY (block_height, tx_pos);


--
-- Name: asset_tags_asset_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY asset_tags
    ADD CONSTRAINT asset_tags_asset_id_key UNIQUE (asset_id);


--
-- Name: assets_alias_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_alias_key UNIQUE (alias);


--
-- Name: assets_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_client_token_key UNIQUE (client_token);


--
-- Name: assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (id);


--
-- Name: blocks_height_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY blocks
    ADD CONSTRAINT blocks_height_key UNIQUE (height);


--
-- Name: blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY blocks
    ADD CONSTRAINT blocks_pkey PRIMARY KEY (block_hash);


--
-- Name: blocks_txs_tx_hash_block_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY blocks_txs
    ADD CONSTRAINT blocks_txs_tx_hash_block_hash_key UNIQUE (tx_hash, block_hash);


--
-- Name: generator_pending_block_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY generator_pending_block
    ADD CONSTRAINT generator_pending_block_pkey PRIMARY KEY (singleton);


--
-- Name: issuance_totals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY issuance_totals
    ADD CONSTRAINT issuance_totals_pkey PRIMARY KEY (asset_id);


--
-- Name: leader_singleton_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY leader
    ADD CONSTRAINT leader_singleton_key UNIQUE (singleton);


--
-- Name: migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (filename);


--
-- Name: mockhsm_alias_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY mockhsm
    ADD CONSTRAINT mockhsm_alias_key UNIQUE (alias);


--
-- Name: mockhsm_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY mockhsm
    ADD CONSTRAINT mockhsm_pkey PRIMARY KEY (xpub);


--
-- Name: pool_txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY pool_txs
    ADD CONSTRAINT pool_txs_pkey PRIMARY KEY (tx_hash);


--
-- Name: pool_txs_sort_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY pool_txs
    ADD CONSTRAINT pool_txs_sort_id_key UNIQUE (sort_id);


--
-- Name: query_blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY query_blocks
    ADD CONSTRAINT query_blocks_pkey PRIMARY KEY (height);


--
-- Name: query_indexes_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY query_indexes
    ADD CONSTRAINT query_indexes_id_key UNIQUE (alias);


--
-- Name: query_indexes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY query_indexes
    ADD CONSTRAINT query_indexes_pkey PRIMARY KEY (id);


--
-- Name: reservations_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY reservations
    ADD CONSTRAINT reservations_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: reservations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY reservations
    ADD CONSTRAINT reservations_pkey PRIMARY KEY (reservation_id);


--
-- Name: signers_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY signers
    ADD CONSTRAINT signers_client_token_key UNIQUE (client_token);


--
-- Name: signers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY signers
    ADD CONSTRAINT signers_pkey PRIMARY KEY (id);


--
-- Name: state_trees_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY snapshots
    ADD CONSTRAINT state_trees_pkey PRIMARY KEY (height);


--
-- Name: txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txs
    ADD CONSTRAINT txs_pkey PRIMARY KEY (tx_hash);


--
-- Name: account_control_programs_control_program_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_control_programs_control_program_idx ON account_control_programs USING btree (control_program);


--
-- Name: account_utxos_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_account_id ON account_utxos USING btree (account_id);


--
-- Name: account_utxos_account_id_asset_id_tx_hash_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_account_id_asset_id_tx_hash_idx ON account_utxos USING btree (account_id, asset_id, tx_hash);


--
-- Name: account_utxos_reservation_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_reservation_id_idx ON account_utxos USING btree (reservation_id);


--
-- Name: annotated_accounts_jsondata_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_accounts_jsondata_idx ON annotated_accounts USING gin (data jsonb_path_ops);


--
-- Name: annotated_assets_jsondata_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_assets_jsondata_idx ON annotated_assets USING gin (data jsonb_path_ops);


--
-- Name: annotated_outputs_jsondata_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_outputs_jsondata_idx ON annotated_outputs USING gin (data jsonb_path_ops);


--
-- Name: annotated_outputs_outpoint_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_outputs_outpoint_idx ON annotated_outputs USING btree (tx_hash, output_index);


--
-- Name: annotated_outputs_timespan_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_outputs_timespan_idx ON annotated_outputs USING gist (timespan);


--
-- Name: annotated_txs_data; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_txs_data ON annotated_txs USING gin (data);


--
-- Name: assets_sort_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX assets_sort_id ON assets USING btree (sort_id);


--
-- Name: blocks_txs_block_height_block_pos_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX blocks_txs_block_height_block_pos_key ON blocks_txs USING btree (block_height, block_pos);


--
-- Name: query_blocks_timestamp_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX query_blocks_timestamp_idx ON query_blocks USING btree ("timestamp");


--
-- Name: reservations_asset_id_account_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX reservations_asset_id_account_id_idx ON reservations USING btree (asset_id, account_id);


--
-- Name: reservations_expiry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX reservations_expiry ON reservations USING btree (expiry);


--
-- Name: signed_blocks_block_height_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX signed_blocks_block_height_idx ON signed_blocks USING btree (block_height);


--
-- Name: signers_type_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX signers_type_id_idx ON signers USING btree (type, id);


--
-- Name: account_utxos_reservation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_reservation_id_fkey FOREIGN KEY (reservation_id) REFERENCES reservations(reservation_id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--

insert into migrations (filename, hash) values ('2016-08-16.0.core.initial-schema.sql', '60cd6638983c225741317bb71385a7e740425f968685a9a8f54db3de4b78fbf5');
insert into migrations (filename, hash) values ('2016-08-16.1.api.add-asset-aliases.sql', '78b8c814db73872e6ebc8c5bcedc342d17f566a0a637470aed2761ae09060873');
insert into migrations (filename, hash) values ('2016-08-16.2.query.rename-index-alias.sql', 'eb1b6b4fa602ac21d7f957f8fda0b94dd50fb710055842cead111a0bb35d93ae');
insert into migrations (filename, hash) values ('2016-08-16.3.api.alias-keys.sql', 'ee7702a963064b004800ee356558ec2a2a3062f443ff7871d1fc2a873f22665e');
insert into migrations (filename, hash) values ('2016-08-17.0.query.index-id.sql', '538ce1a1f61b496d1809049f3934ba445177e5b71af2e802d2fbcb009a8d80cb');
insert into migrations (filename, hash) values ('2016-08-19.0.generator.generated-block.sql', '8068324f63c2d973f0eac120460ca202711bfab0f734b789f18206f21abd3a80');
insert into migrations (filename, hash) values ('2016-08-22.0.account.change-utxos.sql', '11dff1da7353fd6896c4f153654029d717f838b4a4e528cf7139bfcf35ebf124');
insert into migrations (filename, hash) values ('2016-08-23.0.query.sum-by.sql', '82a04d0595f19df735dd35e896496c79a70398a8428131ece4d441aaf2f3836c');
