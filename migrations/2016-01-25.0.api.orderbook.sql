CREATE TABLE orderbook_prices (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    offer_amount bigint NOT NULL,
    payment_amount bigint NOT NULL
);

CREATE TABLE orderbook_utxos (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    seller_id text NOT NULL
);

DROP INDEX utxos_account_id_asset_id_reserved_at_idx;

ALTER TABLE utxos ALTER account_id DROP NOT NULL;

ALTER TABLE ONLY orderbook_utxos
    ADD CONSTRAINT orderbook_utxos_pkey PRIMARY KEY (tx_hash, index);

CREATE INDEX orderbook_prices_asset_id_idx ON orderbook_prices USING btree (asset_id);

CREATE INDEX orderbook_utxos_seller_id_idx ON orderbook_utxos USING btree (seller_id);

CREATE INDEX utxos_account_id_asset_id_reserved_at_idx ON utxos USING btree (account_id, asset_id, reserved_until) WHERE (account_id IS NOT NULL);

ALTER TABLE ONLY orderbook_prices
    ADD CONSTRAINT orderbook_prices_tx_hash_fkey FOREIGN KEY (tx_hash, index) REFERENCES utxos(tx_hash, index) ON DELETE CASCADE;

ALTER TABLE ONLY orderbook_utxos
    ADD CONSTRAINT orderbook_utxos_tx_hash_fkey FOREIGN KEY (tx_hash, index) REFERENCES utxos(tx_hash, index) ON DELETE CASCADE;
