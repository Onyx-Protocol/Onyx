ALTER TABLE wallets ADD COLUMN generated_keys text[] DEFAULT '{}' NOT NULL;
ALTER TABLE asset_groups ADD COLUMN generated_keys text[] DEFAULT '{}' NOT NULL;
