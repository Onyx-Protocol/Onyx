ALTER TABLE account_utxos
	ADD COLUMN confirmed_in int8,
	ADD COLUMN block_pos int4;

UPDATE account_utxos AS a
SET confirmed_in=b.block_height, block_pos=b.block_pos
FROM blocks_txs b
WHERE a.tx_hash = b.tx_hash;
