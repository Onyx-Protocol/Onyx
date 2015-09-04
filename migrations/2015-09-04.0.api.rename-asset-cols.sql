ALTER TABLE assets DROP CONSTRAINT assets_wallet_id_fkey;
ALTER TABLE assets RENAME wallet_id TO asset_group_id;
ALTER TABLE assets RENAME keys TO keyset;
ALTER TABLE assets ADD CONSTRAINT assets_asset_group_id_fkey
	FOREIGN KEY (asset_group_id) REFERENCES asset_groups (id);
ALTER TABLE assets ALTER redeem_script SET NOT NULL;
ALTER TABLE assets ADD COLUMN label text NOT NULL;
