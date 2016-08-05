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
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


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
        ON CONFLICT (account_id, idempotency_key) DO NOTHING
        RETURNING reservations.reservation_id, FALSE AS already_existed, CAST(0 AS BIGINT) AS existing_change INTO row;
    -- Iff the insert was successful, then a row is returned. The IF NOT FOUND check
    -- will be true iff the insert failed because the row already exists.
    IF NOT FOUND THEN
        SELECT r.reservation_id, TRUE AS already_existed, r.change AS existing_change INTO STRICT row
            FROM reservations r
            WHERE r.account_id = inp_account_id AND r.idempotency_key = inp_idempotency_key;
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
    block_timestamp bigint
);


--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE assets (
    id text NOT NULL,
    key_index bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    definition_mutable boolean DEFAULT false NOT NULL,
    definition bytea,
    sort_id text DEFAULT next_chain_id('asset'::text) NOT NULL,
    issuance_program bytea NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    client_token text,
    genesis_hash text NOT NULL,
    signer_id text NOT NULL
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
-- Name: auth_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE auth_tokens (
    id text DEFAULT next_chain_id('at'::text) NOT NULL,
    secret_hash bytea NOT NULL,
    user_id text,
    type text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone
);


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
-- Name: invitations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE invitations (
    id text NOT NULL,
    email text NOT NULL,
    role text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT invitations_role_check CHECK (((role = 'developer'::text) OR (role = 'admin'::text)))
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
    xpub_hash text NOT NULL
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
-- Name: query_indexes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE query_indexes (
    internal_id integer NOT NULL,
    id text NOT NULL,
    type text NOT NULL,
    query text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
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

ALTER SEQUENCE query_indexes_internal_id_seq OWNED BY query_indexes.internal_id;


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
    asset_id text NOT NULL,
    account_id text NOT NULL,
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
    archived boolean DEFAULT false NOT NULL,
    tags jsonb
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
-- Name: state_trees; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE state_trees (
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
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE users (
    id text DEFAULT next_chain_id('u'::text) NOT NULL,
    email text NOT NULL,
    password_hash bytea NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    pwreset_secret_hash bytea,
    pwreset_expires_at timestamp with time zone,
    role text DEFAULT 'developer'::text NOT NULL,
    CONSTRAINT users_role_check CHECK (((role = 'developer'::text) OR (role = 'admin'::text)))
);


--
-- Name: internal_id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY query_indexes ALTER COLUMN internal_id SET DEFAULT nextval('query_indexes_internal_id_seq'::regclass);


--
-- Name: key_index; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY signers ALTER COLUMN key_index SET DEFAULT nextval('signers_key_index_seq'::regclass);


--
-- Name: account_utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_pkey PRIMARY KEY (tx_hash, index);


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
-- Name: invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY invitations
    ADD CONSTRAINT invitations_pkey PRIMARY KEY (id);


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
-- Name: query_indexes_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY query_indexes
    ADD CONSTRAINT query_indexes_id_key UNIQUE (id);


--
-- Name: query_indexes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY query_indexes
    ADD CONSTRAINT query_indexes_pkey PRIMARY KEY (internal_id);


--
-- Name: reservations_account_id_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY reservations
    ADD CONSTRAINT reservations_account_id_idempotency_key_key UNIQUE (account_id, idempotency_key);


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

ALTER TABLE ONLY state_trees
    ADD CONSTRAINT state_trees_pkey PRIMARY KEY (height);


--
-- Name: txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txs
    ADD CONSTRAINT txs_pkey PRIMARY KEY (tx_hash);


--
-- Name: users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


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
-- Name: assets_sort_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX assets_sort_id ON assets USING btree (sort_id);


--
-- Name: auth_tokens_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX auth_tokens_user_id_idx ON auth_tokens USING btree (user_id);


--
-- Name: blocks_txs_block_height_block_pos_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX blocks_txs_block_height_block_pos_key ON blocks_txs USING btree (block_height, block_pos);


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
-- Name: users_lower_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX users_lower_idx ON users USING btree (lower(email));


--
-- Name: account_utxos_reservation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_reservation_id_fkey FOREIGN KEY (reservation_id) REFERENCES reservations(reservation_id) ON DELETE SET NULL;


--
-- Name: auth_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY auth_tokens
    ADD CONSTRAINT auth_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);


--
-- PostgreSQL database dump complete
--

insert into migrations (filename, hash) values ('2016-01-20.0.api.schema-snapshot.sql', '82e95b4385631ffb68e674dc9c48309bac2bb1350e0d9cb7d1599ca5ea7ce83b');
insert into migrations (filename, hash) values ('2016-01-25.0.api.orderbook.sql', '75a159160218af51005eadcc4320549d499dc966fbc37d63039f6c6ddc7cd59e');
insert into migrations (filename, hash) values ('2016-01-27.0.api.archive-items.sql', '17d0e2c3cab3f5f819a88a70d6f3189a56cf14ad0d0466598afbbe5535ca5801');
insert into migrations (filename, hash) values ('2016-02-08.0.txdb.separate-utxos-table.sql', 'af9b21b4cff5327d45135a716af499ee1613741473ec948be1c5a9edb209e377');
insert into migrations (filename, hash) values ('2016-02-12.0.api.dbonly-utxos.sql', '3d6b0347d7b2a27e250b18a99a3a70e47bba1b5d018e864bc71a796a6646b637');
insert into migrations (filename, hash) values ('2016-02-18.0.api.asset-client-token.sql', 'f06be0b26d6445c6140ac7842baffdf933794cbe33eae039259de407c8b57bdb');
insert into migrations (filename, hash) values ('2016-02-19.0.api.accounts-client-token.sql', '838af44c35a9380fcfce6acdc04fe9d08d44de6de51b3fa760634c24c7e3ea5d');
insert into migrations (filename, hash) values ('2016-02-22.0.txdb.signed-blocks.sql', '3c78428297cdc542e80f3d736c77a01779ef86d39c76d95da2dc31f8a0bb4d5f');
insert into migrations (filename, hash) values ('2016-02-23.0.api.blocks-txs-block-pos.sql', 'ec296d93e7560fe487d694b109ae19ff465479ffa403f53ce51d230d56a3cd0b');
insert into migrations (filename, hash) values ('2016-02-23.1.api.backfill-block-pos.go', '65c1a1ca38ad01dd0d302c38fa4178a585a38fe9970014ee5d127a767399807f');
insert into migrations (filename, hash) values ('2016-02-23.3.api.blocks-txs-constraints.sql', 'cd1e6a80cc7d8429314b8925e4acce2cc19f794d33ac98fdb89c74e812beacac');
insert into migrations (filename, hash) values ('2016-02-23.4.api.blocks-txs-sort-index.sql', 'eaa748bd2bf7072e0bc18a411ddd4792e85fc031f126d1347199532b781181ba');
insert into migrations (filename, hash) values ('2016-02-24.5.api.utxos-asset-id-index.sql', 'fcd117dde2bd2684c2130ca45966c6a5717faa3eccc947d1f22456a615c59e7e');
insert into migrations (filename, hash) values ('2016-02-25.0.utxodb.idempotency-key.sql', '1d0b97503128690fffc1bd9f8f29c7542ecd8d6d404ff0719f9319d1dba43d1f');
insert into migrations (filename, hash) values ('2016-02-26.0.api.nodes-client-token.sql', '6ee4a8fc86c631d48bfba95be788926d8847cd457cbb9d56362cec4fac4574e5');
insert into migrations (filename, hash) values ('2016-03-01.0.appdb.noactivity.sql', '06caafe93fdef520c98b333730242df41c448f8fbd802e591358b4c45b5839c8');
insert into migrations (filename, hash) values ('2016-03-01.1.appdb.notxids.sql', 'f218dd07bb3b795365af69d6739ac089cf896daac166df8b895ac77920636aac');
insert into migrations (filename, hash) values ('2016-03-04.0.utxodb.reserve-utxos-by-tx-hash.sql', 'c5530a9ed1aaeb88b0e27367e4f695b84bd5108478b7510368c5a2e293777cd6');
insert into migrations (filename, hash) values ('2016-03-17.0.txdb.add-state-trees.sql', '6265013d655390a89f6c2fa594d2be8caad3e20c8cdf1ea67641b0d907e2ffac');
insert into migrations (filename, hash) values ('2016-03-18.0.api.leader-election.sql', 'a871c63bdc1259405a84d2eb08176e263fea39d00b2372832d64d2a9fabd8eaa');
insert into migrations (filename, hash) values ('2016-03-22.0.api.voting.sql', 'b997861457d0716881b84b2737568506961734579bda392d4331c45a63a2e4eb');
insert into migrations (filename, hash) values ('2016-03-25.0.api.voting-account-id-index.sql', '7240ae32c08b6de39fa9abfb91cbe4323ec7a06a955d7ad29a77162063ee44e3');
insert into migrations (filename, hash) values ('2016-03-29.0.api.voting-void.sql', '9396e28ea8aa057395f1e0a6a939c2a3e3a390137bb00943fb3e39b1e380f58c');
insert into migrations (filename, hash) values ('2016-03-31.0.api.voting-admin.sql', 'cc1b7a4149a7010c281e5e2a0d67a9156c99edce1396f841f79490ab187b6fdd');
insert into migrations (filename, hash) values ('2016-04-19.0.txdb.notifyblocks.sql', '5b080e150c0ac6327f3da5153e37930f7a083e6879cc3cf1532178889d25f631');
insert into migrations (filename, hash) values ('2016-04-20.0.api.voting-tokens.sql', '117b3786ba5e21b1519e8bb850220f765728033643f7458879d4132d8f6332fe');
insert into migrations (filename, hash) values ('2016-04-22.0.txdb.notifyblocks.sql', '07f8cdc28d850c9c7b972ab99f94a69c28aff705d0d516e4c177e0d3a4cbcaf7');
insert into migrations (filename, hash) values ('2016-05-04.0.txdb.nocontracthash.sql', '8828fd884809a321d8e844f8602234f4518cfe5fddbf2a419074ad4b02058455');
insert into migrations (filename, hash) values ('2016-05-17.0.utxodb.reserve-by-index.sql', '4e5b2e5804092c2b32f960ed9112ffeea0809c4c674882089f3e9324f3efe42c');
insert into migrations (filename, hash) values ('2016-05-18.0.appdb.acct-utxos-denorm.sql', '77e6093753e1b1229f336298d33ab0285993a2bc1004ddc631f8937b9d23334d');
insert into migrations (filename, hash) values ('2016-05-18.1.appdb.acct-utxos-denorm-height.sql', '1cad82c9ca061409a362170522bc1bf1bccd18600f8c06d6832cce47c9f4c04b');
insert into migrations (filename, hash) values ('2016-05-18.2.api.voting-right-index.sql', '77fed0a2776e297a51d4472f6ea4fe7c86f987af892c8032c3dc871587cf64bd');
insert into migrations (filename, hash) values ('2016-05-19.0.utxodb.denorm-pool-spent.sql', '84b06cfed624e3ece0cbe8b323f46c48f3799f3a0be9ad68cf6f84f882085f66');
insert into migrations (filename, hash) values ('2016-05-20.0.utxodb.remove-acct-fk.sql', 'd3cb2753b8834d611e4c4ca26dc75609841e3ae938871414c55adea3de0b2236');
insert into migrations (filename, hash) values ('2016-05-23.0.api.drop-voting-option-count.sql', 'fdd3fda84e6b2a0c78456e50790842aade84d4b2a794a6abb166856869aa413c');
insert into migrations (filename, hash) values ('2016-05-24.0.utxodb.denorm-ob.sql', 'f1b2cee039887f705e5dc38ceb055135b51385fb54d7f4b1c72da12d50f22510');
insert into migrations (filename, hash) values ('2016-05-25.0.api.voting-token-lots.sql', 'd997c322e1f41c83cbd5a6c160f6e3171a8b9215e1c213282d209ce9356b899c');
insert into migrations (filename, hash) values ('2016-05-25.1.utxodb.remove-ob-fk.sql', '83d7fb0b72219115bc591c2b261f0d0ab313658e1507ab8a7ce85f313803d0f5');
insert into migrations (filename, hash) values ('2016-05-26.0.appdb.acct-utxos-block-timestamp.sql', 'f4ebe7c313d074caad151cc774fc2ce83aa402f6a1130af42f6295f37f54fc68');
insert into migrations (filename, hash) values ('2016-05-26.1.appdb.backfill-acct-utxos-block-timestamp.go', '319115b880b8eaf6c14d705c1705ba5f6665189561655a35607e664beaee70b4');
insert into migrations (filename, hash) values ('2016-05-26.2.api.add-voting-registration-id.sql', 'e7c1388c31afc622132bfb8927db77700a504abd387afdd454c19fe762bb50c7');
insert into migrations (filename, hash) values ('2016-05-26.3.appdb.historical-outputs.sql', '29a80558a5dde5ed85b1a157221e6441ea73dac718ea5587f92be666ef286d97');
insert into migrations (filename, hash) values ('2016-05-27.0.appdb.historical-metadata.sql', 'e53e310b8861b359086ba6201c17655c98dde8e13f0ba919b1abdc2a5b1822d3');
insert into migrations (filename, hash) values ('2016-05-27.1.txdb.drop-pool-inputs.sql', 'b30335266d56167fffb8577dbd890057b2200b5eb0e0b0a80b9f397c58423c43');
insert into migrations (filename, hash) values ('2016-05-31.0.appdb.optional-accounts-historical-outputs.sql', '1b590e145f7bfd72acae5e78ff33163727b044360541cc1e68453c86e8f5fd0d');
insert into migrations (filename, hash) values ('2016-06-01.0.core.historical-outputs-outpoint-index.sql', 'f805f273b80affc6e0034cb6cd978ba2f6f1d33bee761ccbe1f3c2d06ad09a39');
insert into migrations (filename, hash) values ('2016-06-06.0.txdb.state-tree-values.sql', 'aa69e9db24b5cc612183af35452d9f3ac9d8442674a5e7823ba88c12cb388624');
insert into migrations (filename, hash) values ('2016-06-09.0.txdb.drop-blocks-utxos.sql', '168a86e8699218b502a06b26f7634d85c72677f26570791d5bc26a894d26127f');
insert into migrations (filename, hash) values ('2016-06-09.1.core.rename-explorer-outputs.sql', 'c09b0b75562e5c53852cc84f96f24063eea6b46129f21e8f29b98953e2fb1deb');
insert into migrations (filename, hash) values ('2016-06-09.2.txdb.drop-utxos.sql', 'a6d94f3afeae6145caef7fe5919aaacb5050ce68b312e53461b109ada88e382e');
insert into migrations (filename, hash) values ('2016-06-10.0.voting.change-closed-column.sql', '93487fb8edd612a79da022aa9b17b2a94e121c4facb3199c7edd0ce12248389a');
insert into migrations (filename, hash) values ('2016-06-13.0.txdb.state-tree-snapshots.sql', 'de6000306a774d97fd46a095559643231f0db786ece08a89fb913c087b4babd7');
insert into migrations (filename, hash) values ('2016-06-20.0.voting.remove-deadline.sql', '96d0e78feca6917e329c6dfea7c3a581b98732f303d83c25376cf2727cd8faf6');
insert into migrations (filename, hash) values ('2016-07-07.0.asset.issuance-totals-block.sql', 'be9df7c943c20c8a8231145fa73f6df029eec678a21115682d94e65858372b01');
insert into migrations (filename, hash) values ('2016-07-08.0.core.drop-members.sql', '3fac6b9ad710c286e47aee3748e321902ceca3c80aaf1525c7d0934e762c53aa');
insert into migrations (filename, hash) values ('2016-07-19.0.core.node-txs-unique.sql', 'e8a337edc26bb4c37d97b084f0e3167b8e7900e4dc676b3ffe90544376dc147d');
insert into migrations (filename, hash) values ('2016-07-19.1.asset.drop-spent-in-pool.sql', 'a1543793493f0d100352e66bf27979ee34298459c975fae70d6cb53c49955b67');
insert into migrations (filename, hash) values ('2016-07-21.0.core.asset-genesis-hash.sql', '45642a9fd2d033208f78dafffaaab99fb7e9dfbb7b15224d254b283fa9db416b');
insert into migrations (filename, hash) values ('2016-07-26.0.api.drop-manager-issuer-txs.sql', '2f7b14ce50118f1effcf153bf8c8f5cc7859af45487a383df6a7c8e3f5e8f15b');
insert into migrations (filename, hash) values ('2016-07-27.0.core.drop-smartcontract-tables.sql', '56b09b59392114eff794db343e0a045f8648db77bd086159dba4fe9eafea2dc6');
insert into migrations (filename, hash) values ('2016-07-28.0.core.mockhsm.sql', '4f8c1a90f2789b5db62bdf6cd94255e6e41cce1f78e3254643032d1d6a53438c');
insert into migrations (filename, hash) values ('2016-07-28.1.explorer.drop-explorer-outputs.sql', '99d36e88d57cc4405a0bec300fe6b88675278b9aad91a83d1d1fe50533355adb');
insert into migrations (filename, hash) values ('2016-07-29.0.signer.add-signers.sql', '31585f1d6d2c1cf2f3157929b355e2b81f9da6e117b58c21c47b2ba3f9194a0a');
insert into migrations (filename, hash) values ('2016-08-02.0.query.indexes.sql', '9f50b380a05e7b1d65cf10a8339b5be52aebbcfac1266ef5f55edd312d3b067c');
insert into migrations (filename, hash) values ('2016-08-03.0.assets.use-signers.sql', '5e1d674c4f61f6b2f238e8600b145e44a819827be3a8b79764c432540c49f051');
insert into migrations (filename, hash) values ('2016-08-03.1.core.remove-projects.sql', '801a54a49cdde74e5d8995a91be1ca0d4aa0715374a088fe4d2fe041d19cd09d');
insert into migrations (filename, hash) values ('2016-08-03.2.mockhsm.add-xpub-hash.sql', '70cb6105554a3691c485edcc9472fe4823297fa6fdf0ea7e46bb8dbebe32a076');
insert into migrations (filename, hash) values ('2016-08-03.3.signer.add-tags-to-signers.sql', '96d4fca692e5eedbc7c2a65e993936717cdbb09bd3f0b220ee862cc1aec1b5a9');
insert into migrations (filename, hash) values ('2016-08-04.0.assets.remove-mutable-definitions.sql', '0daf236696d4f80f96c8eeeb062673180991e64d9a27e65e29f45b8fc9564830');
insert into migrations (filename, hash) values ('2016-08-05.0.core.remove-asset-redeem.sql', 'd9f1fe0eeb9b3702fb366586f2b208f1c0eab22a110f49d683a685a1c924da3b');
