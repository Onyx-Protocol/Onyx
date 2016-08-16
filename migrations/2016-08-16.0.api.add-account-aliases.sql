ALTER TABLE account_tags RENAME to accounts;
ALTER TABLE accounts ADD COLUMN alias text, ADD COLUMN archived boolean NOT NULL default false, ADD UNIQUE (alias);
