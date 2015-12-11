ALTER TABLE accounts ADD COLUMN keys text[] DEFAULT '{}'::text[] NOT NULL;
