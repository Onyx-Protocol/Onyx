ALTER TABLE assets
  DROP COLUMN definition,
  ADD COLUMN definition jsonb;
