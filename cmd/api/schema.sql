--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: plv8; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plv8 WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plv8; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plv8 IS 'PL/JavaScript (v8) trusted procedural language';


SET search_path = public, pg_catalog;

--
-- Name: next_chain_id(text); Type: FUNCTION; Schema: public; Owner: -
--

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


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: assets; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE assets (
    id text NOT NULL,
    wallet_id text NOT NULL,
    key_index bigint NOT NULL,
    keys text[] DEFAULT '{}'::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    definition_mutable boolean DEFAULT false NOT NULL,
    definition_url text DEFAULT ''::text NOT NULL,
    definition bytea,
    redeem_script bytea
);


--
-- Name: buckets; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE buckets (
    id text DEFAULT next_chain_id('b'::text) NOT NULL,
    wallet_id text NOT NULL,
    key_index bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    next_receiver_index bigint DEFAULT 0 NOT NULL
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
-- Name: keys; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE keys (
    id text NOT NULL,
    type text,
    xpub text,
    enc_xpriv text,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: outputs; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE outputs (
    txid text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    amount bigint NOT NULL,
    receiver_id text NOT NULL,
    bucket_id text NOT NULL,
    wallet_id text NOT NULL,
    reserved_at timestamp with time zone DEFAULT '1979-12-31 16:00:00-08'::timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: receivers; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE receivers (
    id text DEFAULT next_chain_id('r'::text) NOT NULL,
    wallet_id text NOT NULL,
    bucket_id text NOT NULL,
    keyset text[] NOT NULL,
    key_index bigint NOT NULL,
    address text NOT NULL,
    memo text,
    amount bigint,
    is_change boolean DEFAULT false NOT NULL,
    expiration timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: rotations; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE rotations (
    id text DEFAULT next_chain_id('rot'::text) NOT NULL,
    wallet_id text NOT NULL,
    pek_pub text,
    keyset text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: wallets; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE wallets (
    id text DEFAULT next_chain_id('w'::text) NOT NULL,
    application_id text NOT NULL,
    development boolean,
    block_chain text,
    sigs_required integer DEFAULT 2 NOT NULL,
    chain_keys integer DEFAULT 1 NOT NULL,
    key_index bigint NOT NULL,
    label text NOT NULL,
    current_rotation text NOT NULL,
    pek text NOT NULL,
    next_asset_index bigint DEFAULT 0 NOT NULL,
    next_bucket_index bigint DEFAULT 0 NOT NULL,
    buckets_count bigint DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: wallets_key_index_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE wallets_key_index_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: wallets_key_index_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE wallets_key_index_seq OWNED BY wallets.key_index;


--
-- Name: key_index; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY wallets ALTER COLUMN key_index SET DEFAULT nextval('wallets_key_index_seq'::regclass);


--
-- Name: assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (id);


--
-- Name: buckets_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY buckets
    ADD CONSTRAINT buckets_pkey PRIMARY KEY (id);


--
-- Name: keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY keys
    ADD CONSTRAINT keys_pkey PRIMARY KEY (id);


--
-- Name: outputs_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY outputs
    ADD CONSTRAINT outputs_pkey PRIMARY KEY (txid, index);


--
-- Name: receivers_address_key; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY receivers
    ADD CONSTRAINT receivers_address_key UNIQUE (address);


--
-- Name: rotations_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY rotations
    ADD CONSTRAINT rotations_pkey PRIMARY KEY (id);


--
-- Name: wallets_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY wallets
    ADD CONSTRAINT wallets_pkey PRIMARY KEY (id);


--
-- Name: buckets_wallet_path; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX buckets_wallet_path ON buckets USING btree (wallet_id, key_index);


--
-- Name: outputs_bucket_id_asset_id_reserved_at_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX outputs_bucket_id_asset_id_reserved_at_idx ON outputs USING btree (bucket_id, asset_id, reserved_at);


--
-- Name: outputs_receiver_id_asset_id_reserved_at_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX outputs_receiver_id_asset_id_reserved_at_idx ON outputs USING btree (receiver_id, asset_id, reserved_at);


--
-- Name: outputs_wallet_id_asset_id_reserved_at_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX outputs_wallet_id_asset_id_reserved_at_idx ON outputs USING btree (wallet_id, asset_id, reserved_at);


--
-- Name: receivers_bucket_id_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX receivers_bucket_id_idx ON receivers USING btree (bucket_id);


--
-- Name: receivers_bucket_id_key_index_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX receivers_bucket_id_key_index_idx ON receivers USING btree (bucket_id, key_index);


--
-- Name: receivers_wallet_id_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX receivers_wallet_id_idx ON receivers USING btree (wallet_id);


--
-- Name: wallets_application_id_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX wallets_application_id_idx ON wallets USING btree (application_id);


--
-- Name: assets_wallet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_wallet_id_fkey FOREIGN KEY (wallet_id) REFERENCES wallets(id);


--
-- Name: buckets_wallet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY buckets
    ADD CONSTRAINT buckets_wallet_id_fkey FOREIGN KEY (wallet_id) REFERENCES wallets(id);


--
-- Name: receivers_bucket_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY receivers
    ADD CONSTRAINT receivers_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES buckets(id);


--
-- Name: receivers_wallet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY receivers
    ADD CONSTRAINT receivers_wallet_id_fkey FOREIGN KEY (wallet_id) REFERENCES wallets(id);


--
-- Name: rotations_wallet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY rotations
    ADD CONSTRAINT rotations_wallet_id_fkey FOREIGN KEY (wallet_id) REFERENCES wallets(id);


--
-- PostgreSQL database dump complete
--

