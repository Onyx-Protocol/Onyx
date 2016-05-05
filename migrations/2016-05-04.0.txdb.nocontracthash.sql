DROP INDEX utxos_asset_id_contract_hash_idx;
DROP VIEW IF EXISTS utxos_status;
ALTER TABLE utxos DROP contract_hash;
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
