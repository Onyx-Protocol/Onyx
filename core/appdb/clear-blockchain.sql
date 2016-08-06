-- This file will reset the blockchain data, while leaving
-- platform-specific data (nodes, accounts, etc) intact.
--
-- To run it:
--
--   psql -f clear-blockchain.sql $DBURL

TRUNCATE
	annotated_txs,
	account_utxos,
	issuance_totals,
	account_control_programs,
	pool_txs,
	blocks_txs,
	blocks,
	txs,
	reservations,
	signed_blocks,
	state_trees;

ALTER SEQUENCE address_index_seq RESTART;
ALTER SEQUENCE pool_tx_sort_id_seq RESTART;
