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
	addresses;
