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
	orderbook_prices,
	orderbook_utxos,
	utxos,
	blocks_utxos,
	account_utxos,
	assets,
	issuance_totals,
	addresses,
	pool_inputs,
	pool_txs,
	blocks_txs,
	blocks,
	txs,
	asset_definitions,
	asset_definition_pointers,
	manager_txs_accounts,
	manager_txs,
	issuer_txs_assets,
	issuer_txs,
	reservations,
	signed_blocks;

ALTER SEQUENCE address_index_seq RESTART;
ALTER SEQUENCE pool_tx_sort_id_seq RESTART;
