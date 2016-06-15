BEGIN;

-- Add new height column to reflect the height that issuance totals
-- are up-to-date with.
ALTER TABLE issuance_totals
  DROP COLUMN pool,
  DROP COLUMN destroyed_pool,
  ADD COLUMN height bigint;

ALTER TABLE issuance_totals
  RENAME COLUMN confirmed TO issued;

ALTER TABLE issuance_totals
  RENAME COLUMN destroyed_confirmed TO destroyed;

-- Fill in all existing rows with the current height.
WITH tip_block AS (
    SELECT max(height) AS height FROM blocks
)
UPDATE issuance_totals SET height = (SELECT height FROM tip_block);

-- Add the NON-NULL constraint.
ALTER TABLE issuance_totals
  ALTER COLUMN height SET NOT NULL;

COMMIT;
