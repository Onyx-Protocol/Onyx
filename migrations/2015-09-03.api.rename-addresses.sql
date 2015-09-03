ALTER TABLE receivers RENAME TO addresses;
ALTER TABLE addresses ALTER id SET DEFAULT next_chain_id('addr'::text);

ALTER TABLE buckets RENAME next_receiver_index TO next_address_index;
ALTER TABLE outputs RENAME receiver_id TO address_id;

ALTER INDEX receivers_wallet_id_idx RENAME TO addresses_wallet_id_idx;
ALTER INDEX receivers_bucket_id_key_index_idx RENAME TO addresses_bucket_id_key_index_idx;
ALTER INDEX receivers_bucket_id_idx RENAME TO addresses_bucket_id_idx;
ALTER INDEX receivers_address_key RENAME TO addresses_address_key;
ALTER INDEX outputs_receiver_id_asset_id_reserved_at_idx RENAME TO outputs_address_id_asset_id_reserved_at_idx;

ALTER TABLE addresses RENAME CONSTRAINT receivers_bucket_id_fkey TO addresses_bucket_id_fkey;
ALTER TABLE addresses RENAME CONSTRAINT receivers_wallet_id_fkey TO addresses_wallet_id_fkey;
