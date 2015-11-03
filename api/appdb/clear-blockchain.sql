-- This file will reset the blockchain data, while leaving
-- platform-specific data (projects, accounts, etc) intact.
--
-- To run it:
--
--   psql -f clear-blockchain.sql $DBURL

TRUNCATE
	issuance_activity_assets,
	issuance_activity,
	activity_accounts,
	activity,
	utxos,
	assets,
	addresses,
	pool_outputs,
	pool_inputs,
	pool_txs,
	blocks,
	asset_definitions,
	asset_definition_pointers;

ALTER SEQUENCE address_index_seq RESTART;
ALTER SEQUENCE pool_tx_sort_id_seq RESTART;
