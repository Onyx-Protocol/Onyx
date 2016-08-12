ALTER TABLE reservations
	ALTER COLUMN account_id DROP NOT NULL,
	ALTER COLUMN asset_id DROP NOT NULL,
	DROP CONSTRAINT reservations_account_id_idempotency_key_key,
	ADD CONSTRAINT reservations_idempotency_key_key UNIQUE (idempotency_key);

CREATE OR REPLACE FUNCTION create_reservation(inp_asset_id text, inp_account_id text, inp_expiry timestamp with time zone, inp_idempotency_key text, OUT reservation_id integer, OUT already_existed boolean, OUT existing_change bigint) RETURNS record
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
