DROP INDEX utxos_address_id_asset_id_reserved_at_idx;

ALTER TABLE utxos
	DROP address_id,
	ADD COLUMN addr_index bigint NOT NULL;
