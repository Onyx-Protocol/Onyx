ALTER TABLE signers DROP COLUMN tags;

CREATE TABLE account_tags (
  account_id text NOT NULL,
  tags jsonb,
  PRIMARY KEY(account_id)
);
