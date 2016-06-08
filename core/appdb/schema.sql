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
                  AND NOT spent_in_pool
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


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: account_utxos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_utxos (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    amount bigint NOT NULL,
    manager_node_id text NOT NULL,
    account_id text NOT NULL,
    addr_index bigint NOT NULL,
    reservation_id integer,
    script bytea NOT NULL,
    metadata bytea NOT NULL,
    confirmed_in bigint,
    block_pos integer,
    spent_in_pool boolean DEFAULT false NOT NULL,
    block_timestamp bigint
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE accounts (
    id text DEFAULT next_chain_id('acc'::text) NOT NULL,
    manager_node_id text NOT NULL,
    key_index bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    next_address_index bigint DEFAULT 0 NOT NULL,
    label text,
    keys text[] DEFAULT '{}'::text[] NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    client_token text
);


--
-- Name: address_index_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE address_index_seq
    START WITH 10001
    INCREMENT BY 10000
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: addresses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE addresses (
    id text DEFAULT next_chain_id('a'::text) NOT NULL,
    manager_node_id text NOT NULL,
    account_id text NOT NULL,
    keyset text[] NOT NULL,
    key_index bigint NOT NULL,
    memo text,
    amount bigint,
    is_change boolean DEFAULT false NOT NULL,
    expiration timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    redeem_script bytea NOT NULL,
    pk_script bytea NOT NULL
);


--
-- Name: asset_definition_pointers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE asset_definition_pointers (
    asset_id text NOT NULL,
    asset_definition_hash text NOT NULL
);


--
-- Name: asset_definitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE asset_definitions (
    hash text NOT NULL,
    definition bytea
);


--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE assets (
    id text NOT NULL,
    issuer_node_id text NOT NULL,
    key_index bigint NOT NULL,
    keyset text[] DEFAULT '{}'::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    definition_mutable boolean DEFAULT false NOT NULL,
    definition bytea,
    redeem_script bytea NOT NULL,
    label text NOT NULL,
    sort_id text DEFAULT next_chain_id('asset'::text) NOT NULL,
    inner_asset_id text,
    issuance_script bytea NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    client_token text
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
-- Name: explorer_outputs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE explorer_outputs (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    amount bigint NOT NULL,
    account_id text,
    timespan int8range NOT NULL,
    script bytea DEFAULT '\x'::bytea NOT NULL,
    metadata bytea DEFAULT '\x'::bytea NOT NULL
);


--
-- Name: invitations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE invitations (
    id text NOT NULL,
    project_id text NOT NULL,
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
    pool bigint DEFAULT 0 NOT NULL,
    confirmed bigint DEFAULT 0 NOT NULL,
    destroyed_pool bigint DEFAULT 0 NOT NULL,
    destroyed_confirmed bigint DEFAULT 0 NOT NULL,
    CONSTRAINT issuance_totals_confirmed_check CHECK ((confirmed >= 0)),
    CONSTRAINT issuance_totals_pool_check CHECK ((pool >= 0)),
    CONSTRAINT positive_destroyed_confirmed CHECK ((destroyed_confirmed >= 0)),
    CONSTRAINT positive_destroyed_pool CHECK ((destroyed_pool >= 0))
);


--
-- Name: issuer_nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE issuer_nodes (
    id text DEFAULT next_chain_id('in'::text) NOT NULL,
    project_id text NOT NULL,
    block_chain text DEFAULT 'sandbox'::text NOT NULL,
    sigs_required integer DEFAULT 1 NOT NULL,
    key_index bigint NOT NULL,
    label text NOT NULL,
    keyset text[] NOT NULL,
    next_asset_index bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    generated_keys text[] DEFAULT '{}'::text[] NOT NULL,
    variable_keys integer DEFAULT 0 NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    client_token text
);


--
-- Name: issuer_nodes_key_index_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE issuer_nodes_key_index_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: issuer_nodes_key_index_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE issuer_nodes_key_index_seq OWNED BY issuer_nodes.key_index;


--
-- Name: issuer_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE issuer_txs (
    id text DEFAULT next_chain_id('itx'::text) NOT NULL,
    issuer_node_id text NOT NULL,
    data json NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tx_hash text NOT NULL
);


--
-- Name: issuer_txs_assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE issuer_txs_assets (
    issuer_tx_id text NOT NULL,
    asset_id text NOT NULL
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
-- Name: manager_nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE manager_nodes (
    id text DEFAULT next_chain_id('mn'::text) NOT NULL,
    project_id text NOT NULL,
    block_chain text DEFAULT 'sandbox'::text NOT NULL,
    sigs_required integer DEFAULT 1 NOT NULL,
    key_index bigint NOT NULL,
    label text NOT NULL,
    current_rotation text,
    next_asset_index bigint DEFAULT 0 NOT NULL,
    next_account_index bigint DEFAULT 0 NOT NULL,
    accounts_count bigint DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    generated_keys text[] DEFAULT '{}'::text[] NOT NULL,
    variable_keys integer DEFAULT 0 NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    client_token text
);


--
-- Name: manager_nodes_key_index_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE manager_nodes_key_index_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: manager_nodes_key_index_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE manager_nodes_key_index_seq OWNED BY manager_nodes.key_index;


--
-- Name: manager_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE manager_txs (
    id text DEFAULT next_chain_id('mtx'::text) NOT NULL,
    manager_node_id text NOT NULL,
    data json NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tx_hash text NOT NULL
);


--
-- Name: manager_txs_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE manager_txs_accounts (
    manager_tx_id text NOT NULL,
    account_id text NOT NULL
);


--
-- Name: members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE members (
    project_id text NOT NULL,
    user_id text NOT NULL,
    role text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT members_role_check CHECK (((role = 'developer'::text) OR (role = 'admin'::text)))
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
-- Name: orderbook_prices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE orderbook_prices (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    offer_amount bigint NOT NULL,
    payment_amount bigint NOT NULL
);


--
-- Name: orderbook_utxos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE orderbook_utxos (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    seller_id text NOT NULL,
    asset_id text NOT NULL,
    amount bigint NOT NULL,
    script bytea NOT NULL
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
-- Name: projects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE projects (
    id text DEFAULT next_chain_id('proj'::text) NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    archived boolean DEFAULT false NOT NULL
);


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
-- Name: rotations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE rotations (
    id text DEFAULT next_chain_id('rot'::text) NOT NULL,
    manager_node_id text NOT NULL,
    keyset text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: signed_blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE signed_blocks (
    block_height bigint NOT NULL,
    block_hash text NOT NULL
);


--
-- Name: state_trees; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE state_trees (
    key text NOT NULL,
    hash text NOT NULL,
    leaf boolean NOT NULL,
    value bytea
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
    pwreset_expires_at timestamp with time zone
);


--
-- Name: utxos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE utxos (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    amount bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata bytea DEFAULT '\x'::bytea NOT NULL,
    script bytea DEFAULT '\x'::bytea NOT NULL
);


--
-- Name: voting_rights; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE voting_rights (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    account_id text,
    holder bytea NOT NULL,
    deadline bigint,
    delegatable boolean NOT NULL,
    ownership_chain bytea NOT NULL,
    block_height bigint NOT NULL,
    admin_script bytea NOT NULL,
    void_block_height integer,
    ordinal integer NOT NULL
);


--
-- Name: voting_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE voting_tokens (
    asset_id text NOT NULL,
    right_asset_id text NOT NULL,
    tx_hash text NOT NULL,
    index integer NOT NULL,
    state smallint NOT NULL,
    closed boolean NOT NULL,
    vote smallint NOT NULL,
    admin_script bytea NOT NULL,
    amount bigint NOT NULL,
    block_height integer NOT NULL,
    registration_id bytea DEFAULT '\x'::bytea NOT NULL
);


--
-- Name: key_index; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY issuer_nodes ALTER COLUMN key_index SET DEFAULT nextval('issuer_nodes_key_index_seq'::regclass);


--
-- Name: key_index; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY manager_nodes ALTER COLUMN key_index SET DEFAULT nextval('manager_nodes_key_index_seq'::regclass);


--
-- Name: account_utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_pkey PRIMARY KEY (tx_hash, index);


--
-- Name: accounts_manager_node_id_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT accounts_manager_node_id_client_token_key UNIQUE (manager_node_id, client_token);


--
-- Name: accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: addresses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY addresses
    ADD CONSTRAINT addresses_pkey PRIMARY KEY (id);


--
-- Name: asset_definition_pointers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY asset_definition_pointers
    ADD CONSTRAINT asset_definition_pointers_pkey PRIMARY KEY (asset_id);


--
-- Name: asset_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY asset_definitions
    ADD CONSTRAINT asset_definitions_pkey PRIMARY KEY (hash);


--
-- Name: assets_issuer_node_id_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_issuer_node_id_client_token_key UNIQUE (issuer_node_id, client_token);


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
-- Name: issuer_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY issuer_nodes
    ADD CONSTRAINT issuer_nodes_pkey PRIMARY KEY (id);


--
-- Name: issuer_nodes_project_id_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY issuer_nodes
    ADD CONSTRAINT issuer_nodes_project_id_client_token_key UNIQUE (project_id, client_token);


--
-- Name: issuer_txs_add_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY issuer_txs
    ADD CONSTRAINT issuer_txs_add_pkey PRIMARY KEY (id);


--
-- Name: leader_singleton_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY leader
    ADD CONSTRAINT leader_singleton_key UNIQUE (singleton);


--
-- Name: manager_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY manager_nodes
    ADD CONSTRAINT manager_nodes_pkey PRIMARY KEY (id);


--
-- Name: manager_nodes_project_id_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY manager_nodes
    ADD CONSTRAINT manager_nodes_project_id_client_token_key UNIQUE (project_id, client_token);


--
-- Name: manager_txs_add_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY manager_txs
    ADD CONSTRAINT manager_txs_add_pkey PRIMARY KEY (id);


--
-- Name: members_project_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY members
    ADD CONSTRAINT members_project_id_user_id_key UNIQUE (project_id, user_id);


--
-- Name: migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (filename);


--
-- Name: orderbook_utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY orderbook_utxos
    ADD CONSTRAINT orderbook_utxos_pkey PRIMARY KEY (tx_hash, index);


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
-- Name: projects_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);


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
-- Name: rotations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY rotations
    ADD CONSTRAINT rotations_pkey PRIMARY KEY (id);


--
-- Name: state_trees_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY state_trees
    ADD CONSTRAINT state_trees_pkey PRIMARY KEY (key);


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
-- Name: utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY utxos
    ADD CONSTRAINT utxos_pkey PRIMARY KEY (tx_hash, index);


--
-- Name: voting_right_txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY voting_rights
    ADD CONSTRAINT voting_right_txs_pkey PRIMARY KEY (asset_id, ordinal);


--
-- Name: voting_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY voting_tokens
    ADD CONSTRAINT voting_tokens_pkey PRIMARY KEY (tx_hash, index);


--
-- Name: account_utxos_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_account_id ON account_utxos USING btree (account_id);


--
-- Name: account_utxos_account_id_asset_id_tx_hash_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_account_id_asset_id_tx_hash_idx ON account_utxos USING btree (account_id, asset_id, tx_hash);


--
-- Name: account_utxos_manager_node_id_asset_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_manager_node_id_asset_id_idx ON account_utxos USING btree (manager_node_id, asset_id);


--
-- Name: account_utxos_reservation_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_reservation_id_idx ON account_utxos USING btree (reservation_id);


--
-- Name: accounts_manager_node_path; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX accounts_manager_node_path ON accounts USING btree (manager_node_id, key_index);


--
-- Name: addresses_account_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX addresses_account_id_idx ON addresses USING btree (account_id);


--
-- Name: addresses_account_id_key_index_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX addresses_account_id_key_index_idx ON addresses USING btree (account_id, key_index);


--
-- Name: addresses_manager_node_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX addresses_manager_node_id_idx ON addresses USING btree (manager_node_id);


--
-- Name: addresses_pk_script_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX addresses_pk_script_idx ON addresses USING btree (pk_script);


--
-- Name: assets_issuer_node_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX assets_issuer_node_id_idx ON assets USING btree (issuer_node_id);


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
-- Name: explorer_outputs_account_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX explorer_outputs_account_id_idx ON explorer_outputs USING btree (account_id) WHERE (account_id IS NOT NULL);


--
-- Name: explorer_outputs_asset_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX explorer_outputs_asset_id ON explorer_outputs USING btree (asset_id);


--
-- Name: explorer_outputs_timespan_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX explorer_outputs_timespan_idx ON explorer_outputs USING gist (timespan);


--
-- Name: explorer_outputs_tx_hash_index_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX explorer_outputs_tx_hash_index_idx ON explorer_outputs USING btree (tx_hash, index);


--
-- Name: issuer_nodes_project_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issuer_nodes_project_id_idx ON issuer_nodes USING btree (project_id);


--
-- Name: issuer_txs_assets_asset_id_issuer_tx_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issuer_txs_assets_asset_id_issuer_tx_id_idx ON issuer_txs_assets USING btree (asset_id, issuer_tx_id DESC);


--
-- Name: issuer_txs_issuer_node_id_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issuer_txs_issuer_node_id_id_idx ON issuer_txs USING btree (issuer_node_id, id DESC);


--
-- Name: manager_nodes_project_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX manager_nodes_project_id_idx ON manager_nodes USING btree (project_id);


--
-- Name: manager_txs_accounts_account_id_manager_tx_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX manager_txs_accounts_account_id_manager_tx_id_idx ON manager_txs_accounts USING btree (account_id, manager_tx_id DESC);


--
-- Name: manager_txs_manager_node_id_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX manager_txs_manager_node_id_id_idx ON manager_txs USING btree (manager_node_id, id DESC);


--
-- Name: members_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX members_user_id_idx ON members USING btree (user_id);


--
-- Name: orderbook_prices_asset_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX orderbook_prices_asset_id_idx ON orderbook_prices USING btree (asset_id);


--
-- Name: orderbook_utxos_seller_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX orderbook_utxos_seller_id_idx ON orderbook_utxos USING btree (seller_id);


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
-- Name: users_lower_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX users_lower_idx ON users USING btree (lower(email));


--
-- Name: utxos_asset_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX utxos_asset_id_idx ON utxos USING btree (asset_id);


--
-- Name: voting_right_txs_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX voting_right_txs_account_id ON voting_rights USING btree (account_id);


--
-- Name: account_utxos_reservation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_reservation_id_fkey FOREIGN KEY (reservation_id) REFERENCES reservations(reservation_id) ON DELETE SET NULL;


--
-- Name: accounts_manager_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT accounts_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id);


--
-- Name: addresses_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY addresses
    ADD CONSTRAINT addresses_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id);


--
-- Name: addresses_manager_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY addresses
    ADD CONSTRAINT addresses_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id);


--
-- Name: assets_issuer_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_issuer_node_id_fkey FOREIGN KEY (issuer_node_id) REFERENCES issuer_nodes(id);


--
-- Name: auth_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY auth_tokens
    ADD CONSTRAINT auth_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);


--
-- Name: invitations_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY invitations
    ADD CONSTRAINT invitations_project_id_fkey FOREIGN KEY (project_id) REFERENCES projects(id);


--
-- Name: manager_nodes_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY manager_nodes
    ADD CONSTRAINT manager_nodes_project_id_fkey FOREIGN KEY (project_id) REFERENCES projects(id);


--
-- Name: members_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY members
    ADD CONSTRAINT members_project_id_fkey FOREIGN KEY (project_id) REFERENCES projects(id);


--
-- Name: members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY members
    ADD CONSTRAINT members_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);


--
-- Name: rotations_manager_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY rotations
    ADD CONSTRAINT rotations_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id) ON DELETE CASCADE;


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
insert into migrations (filename, hash) values ('2016-05-26.1.appdb.backfill-acct-utxos-block-timestamp.go', 'd2dd1eac7d3d744fe77db8c579db7784b0031c3d8db88d3414a88728a5d0acf5');
insert into migrations (filename, hash) values ('2016-05-26.2.api.add-voting-registration-id.sql', 'e7c1388c31afc622132bfb8927db77700a504abd387afdd454c19fe762bb50c7');
insert into migrations (filename, hash) values ('2016-05-26.3.appdb.historical-outputs.sql', '29a80558a5dde5ed85b1a157221e6441ea73dac718ea5587f92be666ef286d97');
insert into migrations (filename, hash) values ('2016-05-27.0.appdb.historical-metadata.sql', 'e53e310b8861b359086ba6201c17655c98dde8e13f0ba919b1abdc2a5b1822d3');
insert into migrations (filename, hash) values ('2016-05-27.1.txdb.drop-pool-inputs.sql', 'b30335266d56167fffb8577dbd890057b2200b5eb0e0b0a80b9f397c58423c43');
insert into migrations (filename, hash) values ('2016-05-31.0.appdb.optional-accounts-historical-outputs.sql', '1b590e145f7bfd72acae5e78ff33163727b044360541cc1e68453c86e8f5fd0d');
insert into migrations (filename, hash) values ('2016-06-01.0.core.historical-outputs-outpoint-index.sql', 'f805f273b80affc6e0034cb6cd978ba2f6f1d33bee761ccbe1f3c2d06ad09a39');
insert into migrations (filename, hash) values ('2016-06-06.0.txdb.state-tree-values.sql', 'aa69e9db24b5cc612183af35452d9f3ac9d8442674a5e7823ba88c12cb388624');
insert into migrations (filename, hash) values ('2016-06-09.0.txdb.drop-blocks-utxos.sql', '168a86e8699218b502a06b26f7634d85c72677f26570791d5bc26a894d26127f');
insert into migrations (filename, hash) values ('2016-06-09.1.core.rename-explorer-outputs.sql', 'c09b0b75562e5c53852cc84f96f24063eea6b46129f21e8f29b98953e2fb1deb');
