AlTER TABLE voting_tokens
  DROP COLUMN closed,
  ADD COLUMN admin_state smallint NOT NULL;
