-- This file will reset the blockchain data, while leaving
-- platform-specific data (projects, accounts, etc) intact.
--
-- To run it:
--
--   psql -f clear-blockchain.sql $DBURL

TRUNCATE
	orderbook_prices,
	orderbook_utxos,
	explorer_outputs,
	utxos,
	account_utxos,
	issuance_totals,
	addresses,
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
	signed_blocks,
	state_trees,
	voting_rights,
	voting_tokens;

ALTER SEQUENCE address_index_seq RESTART;
ALTER SEQUENCE pool_tx_sort_id_seq RESTART;
