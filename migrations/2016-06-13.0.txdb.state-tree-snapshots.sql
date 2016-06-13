TRUNCATE TABLE state_trees;
ALTER TABLE state_trees
  DROP COLUMN key,
  DROP COLUMN hash,
  DROP COLUMN leaf,
  DROP COLUMN value,
  ADD COLUMN height bigint NOT NULL,
  ADD COLUMN data bytea NOT NULL,
  ADD PRIMARY KEY(height);
