ALTER TABLE assets
	DROP COLUMN issuer_node_id,
	DROP COLUMN keyset,
	DROP COLUMN label,
	DROP COLUMN inner_asset_id,
	ADD COLUMN signer_id text NOT NULL;
ALTER TABLE assets RENAME COLUMN issuance_script TO issuance_program;
ALTER TABLE assets RENAME COLUMN redeem_script TO redeem_program;
ALTER TABLE assets ADD CONSTRAINT assets_client_token_key UNIQUE (client_token);
DROP TABLE issuer_nodes;
