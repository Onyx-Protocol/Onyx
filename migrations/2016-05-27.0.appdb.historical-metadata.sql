ALTER TABLE historical_account_outputs
  ADD COLUMN script bytea,
  ADD COLUMN metadata bytea,
  ALTER COLUMN script SET NOT NULL,
  ALTER COLUMN metadata SET NOT NULL,
  ALTER COLUMN script SET DEFAULT '\x'::bytea,
  ALTER COLUMN metadata SET DEFAULT '\x'::bytea;
