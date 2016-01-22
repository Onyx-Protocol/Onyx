--
-- Name: cancel_reservation(integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE OR REPLACE FUNCTION cancel_reservation(inp_reservation_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM reservations WHERE reservation_id = inp_reservation_id;
END;
$$;


CREATE FUNCTION create_reservation(inp_asset_id text, inp_account_id text, inp_expiry timestamp with time zone) RETURNS integer
    LANGUAGE plpgsql
    AS $$
DECLARE
    new_reservation_id INT;
    row RECORD;
BEGIN
    SELECT NEXTVAL('reservation_seq') INTO STRICT new_reservation_id;
    INSERT INTO reservations (reservation_id, asset_id, account_id, expiry) VALUES (new_reservation_id, inp_asset_id, inp_account_id, inp_expiry);
    RETURN new_reservation_id;
END;
$$;


--
-- Name: expire_reservations(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE OR REPLACE FUNCTION expire_reservations() RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM reservations WHERE expiry < CURRENT_TIMESTAMP;
END;
$$;

--
-- Name: reserve_utxos(text, text, bigint, timestamp with time zone); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION reserve_utxos(inp_asset_id text, inp_account_id text, inp_amt bigint, inp_expiry timestamp with time zone) RETURNS record
    LANGUAGE plpgsql
    AS $$
DECLARE
    row RECORD;
    ret RECORD;
    new_reservation_id INT;
    available BIGINT := 0;
    unavailable BIGINT := 0;
BEGIN
    SELECT create_reservation(inp_asset_id, inp_account_id, inp_expiry) INTO STRICT new_reservation_id;

    LOOP
        SELECT tx_hash, index, amount INTO row
            FROM account_utxos u
            WHERE asset_id = inp_asset_id
                  AND inp_account_id = account_id
                  AND reservation_id IS NULL
                  AND (tx_hash, index) NOT IN (TABLE pool_inputs)
            LIMIT 1
            FOR UPDATE
            SKIP LOCKED;
        IF FOUND THEN
            UPDATE account_utxos SET reservation_id = new_reservation_id
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
        PERFORM cancel_reservation(new_reservation_id);
        new_reservation_id := 0;
    ELSE
        UPDATE reservations SET change = available - inp_amt
            WHERE reservation_id = new_reservation_id;
    END IF;

    SELECT new_reservation_id, available, (available+unavailable < inp_amt) INTO ret;
    RETURN ret;
END;
$$;

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
    asset_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    expiry timestamp with time zone DEFAULT '1970-01-01'::timestamp with time zone NOT NULL,
    change BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY(reservation_id)
);

ALTER TABLE account_utxos
  DROP reserved_until,
  ADD reservation_id INT REFERENCES reservations ON DELETE SET NULL;

CREATE INDEX reservations_asset_id_account_id_idx ON reservations (asset_id, account_id);
CREATE INDEX reservations_expiry ON reservations USING btree (expiry);
CREATE INDEX account_utxos_reservation_id_idx ON account_utxos USING btree (reservation_id);
