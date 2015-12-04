ALTER TABLE pool_outputs ADD contract_hash TEXT;
ALTER TABLE utxos ADD contract_hash TEXT;
CREATE INDEX pool_outputs_asset_id_contract_hash_idx ON pool_outputs USING btree (asset_id, contract_hash) WHERE contract_hash IS NOT NULL;
CREATE INDEX utxos_asset_id_contract_hash_idx ON utxos USING btree (asset_id, contract_hash) WHERE contract_hash IS NOT NULL;
