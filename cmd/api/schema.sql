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

SET default_tablespace = '';

SET default_with_oids = false;

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
    reserved_at timestamp with time zone DEFAULT '1980-01-01 00:00:00-00'::timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: outputs_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace:
--

ALTER TABLE ONLY outputs
    ADD CONSTRAINT outputs_pkey PRIMARY KEY (txid, index);


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
-- PostgreSQL database dump complete
--
