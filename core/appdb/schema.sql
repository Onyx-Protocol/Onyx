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



--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
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
                  AND (tx_hash, index) NOT IN (TABLE pool_inputs)
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
    reservation_id integer
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
-- Name: blocks_utxos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE blocks_utxos (
    tx_hash text NOT NULL,
    index integer NOT NULL
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
    seller_id text NOT NULL
);


--
-- Name: pool_inputs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE pool_inputs (
    tx_hash text NOT NULL,
    index integer NOT NULL
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
    leaf boolean NOT NULL
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
-- Name: utxos_status; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW utxos_status AS
 SELECT u.tx_hash,
    u.index,
    u.asset_id,
    u.amount,
    u.created_at,
    u.metadata,
    u.script,
    (b.tx_hash IS NOT NULL) AS confirmed
   FROM (utxos u
     LEFT JOIN blocks_utxos b ON (((u.tx_hash = b.tx_hash) AND (u.index = b.index))));


--
-- Name: voting_right_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE voting_right_txs (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    account_id text,
    holder bytea NOT NULL,
    deadline bigint,
    delegatable boolean NOT NULL,
    ownership_chain bytea NOT NULL,
    block_height bigint NOT NULL,
    block_tx_index integer NOT NULL,
    void boolean DEFAULT false NOT NULL,
    admin_script bytea NOT NULL
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
    option_count integer NOT NULL,
    secret_hash text NOT NULL,
    admin_script bytea NOT NULL,
    amount bigint NOT NULL
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
-- Name: blocks_utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY blocks_utxos
    ADD CONSTRAINT blocks_utxos_pkey PRIMARY KEY (tx_hash, index);


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
-- Name: pool_inputs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY pool_inputs
    ADD CONSTRAINT pool_inputs_pkey PRIMARY KEY (tx_hash, index);


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

ALTER TABLE ONLY voting_right_txs
    ADD CONSTRAINT voting_right_txs_pkey PRIMARY KEY (tx_hash, index);


--
-- Name: voting_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY voting_tokens
    ADD CONSTRAINT voting_tokens_pkey PRIMARY KEY (asset_id, right_asset_id);


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

CREATE INDEX voting_right_txs_account_id ON voting_right_txs USING btree (account_id);


--
-- Name: account_utxos_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_fkey FOREIGN KEY (tx_hash, index) REFERENCES utxos(tx_hash, index) ON DELETE CASCADE;


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
-- Name: blocks_utxos_tx_hash_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY blocks_utxos
    ADD CONSTRAINT blocks_utxos_tx_hash_fkey FOREIGN KEY (tx_hash, index) REFERENCES utxos(tx_hash, index) ON DELETE CASCADE;


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
-- Name: orderbook_prices_tx_hash_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY orderbook_prices
    ADD CONSTRAINT orderbook_prices_tx_hash_fkey FOREIGN KEY (tx_hash, index) REFERENCES utxos(tx_hash, index) ON DELETE CASCADE;


--
-- Name: orderbook_utxos_tx_hash_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY orderbook_utxos
    ADD CONSTRAINT orderbook_utxos_tx_hash_fkey FOREIGN KEY (tx_hash, index) REFERENCES utxos(tx_hash, index) ON DELETE CASCADE;


--
-- Name: rotations_manager_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY rotations
    ADD CONSTRAINT rotations_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

