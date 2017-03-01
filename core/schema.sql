--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.6
-- Dumped by pg_dump version 9.5.6

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
-- Name: access_token_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE access_token_type AS ENUM (
    'client',
    'network'
);


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


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: access_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE access_tokens (
    id text NOT NULL,
    sort_id text DEFAULT next_chain_id('at'::text),
    type access_token_type NOT NULL,
    hashed_secret bytea NOT NULL,
    created timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: account_control_program_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE account_control_program_seq
    START WITH 10001
    INCREMENT BY 10000
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: account_control_programs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_control_programs (
    signer_id text NOT NULL,
    key_index bigint NOT NULL,
    control_program bytea NOT NULL,
    change boolean NOT NULL,
    expires_at timestamp with time zone
);


--
-- Name: account_utxos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_utxos (
    asset_id bytea NOT NULL,
    amount bigint NOT NULL,
    account_id text NOT NULL,
    control_program_index bigint NOT NULL,
    control_program bytea NOT NULL,
    confirmed_in bigint NOT NULL,
    output_id bytea NOT NULL
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE accounts (
    account_id text NOT NULL,
    tags jsonb,
    alias text
);


--
-- Name: annotated_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_accounts (
    id text NOT NULL,
    alias text NOT NULL,
    keys jsonb NOT NULL,
    quorum integer NOT NULL,
    tags jsonb NOT NULL
);


--
-- Name: annotated_assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_assets (
    id bytea NOT NULL,
    sort_id text NOT NULL,
    alias text NOT NULL,
    issuance_program bytea NOT NULL,
    keys jsonb NOT NULL,
    quorum integer NOT NULL,
    definition jsonb NOT NULL,
    tags jsonb NOT NULL,
    local boolean NOT NULL
);


--
-- Name: annotated_inputs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_inputs (
    tx_hash bytea NOT NULL,
    index integer NOT NULL,
    type text NOT NULL,
    asset_id bytea NOT NULL,
    asset_alias text NOT NULL,
    asset_definition jsonb NOT NULL,
    asset_tags jsonb NOT NULL,
    asset_local boolean NOT NULL,
    amount bigint NOT NULL,
    account_id text,
    account_alias text,
    account_tags jsonb,
    issuance_program bytea NOT NULL,
    reference_data jsonb NOT NULL,
    local boolean NOT NULL,
    spent_output_id bytea NOT NULL
);


--
-- Name: annotated_outputs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_outputs (
    block_height bigint NOT NULL,
    tx_pos integer NOT NULL,
    output_index integer NOT NULL,
    tx_hash bytea NOT NULL,
    timespan int8range NOT NULL,
    output_id bytea NOT NULL,
    type text NOT NULL,
    purpose text NOT NULL,
    asset_id bytea NOT NULL,
    asset_alias text NOT NULL,
    asset_definition jsonb NOT NULL,
    asset_tags jsonb NOT NULL,
    asset_local boolean NOT NULL,
    amount bigint NOT NULL,
    account_id text,
    account_alias text,
    account_tags jsonb,
    control_program bytea NOT NULL,
    reference_data jsonb NOT NULL,
    local boolean NOT NULL
);


--
-- Name: annotated_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE annotated_txs (
    block_height bigint NOT NULL,
    tx_pos integer NOT NULL,
    tx_hash bytea NOT NULL,
    data jsonb NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    block_id bytea NOT NULL,
    local boolean NOT NULL,
    reference_data jsonb NOT NULL
);


--
-- Name: asset_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE asset_tags (
    asset_id bytea NOT NULL,
    tags jsonb
);


--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE assets (
    id bytea NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sort_id text DEFAULT next_chain_id('asset'::text) NOT NULL,
    issuance_program bytea NOT NULL,
    client_token text,
    initial_block_hash bytea NOT NULL,
    signer_id text,
    definition bytea NOT NULL,
    alias text,
    first_block_height bigint,
    vm_version bigint NOT NULL
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
-- Name: block_processors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE block_processors (
    name text NOT NULL,
    height bigint DEFAULT 0 NOT NULL
);


--
-- Name: blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE blocks (
    block_hash bytea NOT NULL,
    height bigint NOT NULL,
    data bytea NOT NULL,
    header bytea NOT NULL
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
-- Name: config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE config (
    singleton boolean DEFAULT true NOT NULL,
    is_signer boolean,
    is_generator boolean,
    blockchain_id bytea NOT NULL,
    configured_at timestamp with time zone NOT NULL,
    generator_url text DEFAULT ''::text NOT NULL,
    block_pub text DEFAULT ''::text NOT NULL,
    remote_block_signers bytea DEFAULT '\x'::bytea NOT NULL,
    generator_access_token text DEFAULT ''::text NOT NULL,
    max_issuance_window_ms bigint,
    id text NOT NULL,
    block_hsm_url text DEFAULT ''::text,
    block_hsm_access_token text DEFAULT ''::text,
    CONSTRAINT config_singleton CHECK (singleton)
);


--
-- Name: generator_pending_block; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE generator_pending_block (
    singleton boolean DEFAULT true NOT NULL,
    data bytea NOT NULL,
    CONSTRAINT generator_pending_block_singleton CHECK (singleton)
);


--
-- Name: leader; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE leader (
    singleton boolean DEFAULT true NOT NULL,
    leader_key text NOT NULL,
    expiry timestamp with time zone DEFAULT '1970-01-01 00:00:00-08'::timestamp with time zone NOT NULL,
    address text NOT NULL,
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
-- Name: mockhsm_sort_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE mockhsm_sort_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: mockhsm; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE mockhsm (
    pub bytea NOT NULL,
    prv bytea NOT NULL,
    alias text,
    sort_id bigint DEFAULT nextval('mockhsm_sort_id_seq'::regclass) NOT NULL,
    key_type text DEFAULT 'chain_kd'::text NOT NULL
);


--
-- Name: query_blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE query_blocks (
    height bigint NOT NULL,
    "timestamp" bigint NOT NULL
);


--
-- Name: signed_blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE signed_blocks (
    block_height bigint NOT NULL,
    block_hash bytea NOT NULL
);


--
-- Name: signers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE signers (
    id text NOT NULL,
    type text NOT NULL,
    key_index bigint NOT NULL,
    quorum integer NOT NULL,
    client_token text,
    xpubs bytea[] NOT NULL
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
    data bytea NOT NULL,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: submitted_txs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE submitted_txs (
    tx_hash bytea NOT NULL,
    height bigint NOT NULL,
    submitted_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: txfeeds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE txfeeds (
    id text DEFAULT next_chain_id('cur'::text) NOT NULL,
    alias text,
    filter text,
    after text,
    client_token text
);


--
-- Name: key_index; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY signers ALTER COLUMN key_index SET DEFAULT nextval('signers_key_index_seq'::regclass);


--
-- Name: access_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_pkey PRIMARY KEY (id);


--
-- Name: account_control_programs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_control_programs
    ADD CONSTRAINT account_control_programs_pkey PRIMARY KEY (control_program);


--
-- Name: account_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT account_tags_pkey PRIMARY KEY (account_id);


--
-- Name: account_utxos_output_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_output_id_key UNIQUE (output_id);


--
-- Name: account_utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_utxos
    ADD CONSTRAINT account_utxos_pkey PRIMARY KEY (output_id);


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
-- Name: annotated_inputs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY annotated_inputs
    ADD CONSTRAINT annotated_inputs_pkey PRIMARY KEY (tx_hash, index);


--
-- Name: annotated_outputs_output_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY annotated_outputs
    ADD CONSTRAINT annotated_outputs_output_id_key UNIQUE (output_id);


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
-- Name: block_processors_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY block_processors
    ADD CONSTRAINT block_processors_name_key UNIQUE (name);


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
-- Name: config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY config
    ADD CONSTRAINT config_pkey PRIMARY KEY (singleton);


--
-- Name: generator_pending_block_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY generator_pending_block
    ADD CONSTRAINT generator_pending_block_pkey PRIMARY KEY (singleton);


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
    ADD CONSTRAINT mockhsm_pkey PRIMARY KEY (pub);


--
-- Name: query_blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY query_blocks
    ADD CONSTRAINT query_blocks_pkey PRIMARY KEY (height);


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
-- Name: sort_id_index; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY mockhsm
    ADD CONSTRAINT sort_id_index UNIQUE (sort_id);


--
-- Name: state_trees_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY snapshots
    ADD CONSTRAINT state_trees_pkey PRIMARY KEY (height);


--
-- Name: submitted_txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY submitted_txs
    ADD CONSTRAINT submitted_txs_pkey PRIMARY KEY (tx_hash);


--
-- Name: txfeeds_alias_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txfeeds
    ADD CONSTRAINT txfeeds_alias_key UNIQUE (alias);


--
-- Name: txfeeds_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txfeeds
    ADD CONSTRAINT txfeeds_client_token_key UNIQUE (client_token);


--
-- Name: txfeeds_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txfeeds
    ADD CONSTRAINT txfeeds_pkey PRIMARY KEY (id);


--
-- Name: account_utxos_asset_id_account_id_confirmed_in_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_utxos_asset_id_account_id_confirmed_in_idx ON account_utxos USING btree (asset_id, account_id, confirmed_in);


--
-- Name: annotated_assets_sort_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_assets_sort_id ON annotated_assets USING btree (sort_id);


--
-- Name: annotated_outputs_timespan_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_outputs_timespan_idx ON annotated_outputs USING gist (timespan);


--
-- Name: annotated_txs_data_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX annotated_txs_data_idx ON annotated_txs USING gin (data jsonb_path_ops);


--
-- Name: assets_sort_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX assets_sort_id ON assets USING btree (sort_id);


--
-- Name: query_blocks_timestamp_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX query_blocks_timestamp_idx ON query_blocks USING btree ("timestamp");


--
-- Name: signed_blocks_block_height_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX signed_blocks_block_height_idx ON signed_blocks USING btree (block_height);


--
-- Name: signers_type_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX signers_type_id_idx ON signers USING btree (type, id);


--
-- PostgreSQL database dump complete
--

insert into migrations (filename, hash) values ('2017-02-03.0.core.schema-snapshot.sql', '1d55668affe0be9f3c19ead9d67bc75cfd37ec430651434d0f2af2706d9f08cd');
insert into migrations (filename, hash) values ('2017-02-07.0.query.non-null-alias.sql', '17028a0bdbc95911e299dc65fe641184e54c87a0d07b3c576d62d023b9a8defc');
insert into migrations (filename, hash) values ('2017-02-16.0.query.spent-output.sql', '7cd52095b6f202d7a25ffe666b7b7d60e7700d314a7559b911e236b72661a738');
insert into migrations (filename, hash) values ('2017-02-28.0.core.remove-outpoints.sql', '067638e2a826eac70d548f2d6bb234660f3200064072baf42db741456ecf8deb');
