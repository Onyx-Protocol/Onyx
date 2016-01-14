ALTER TABLE issuance_totals
    ADD COLUMN destroyed_pool bigint DEFAULT 0 NOT NULL,
    ADD COLUMN destroyed_confirmed bigint DEFAULT 0 NOT NULL;

ALTER TABLE issuance_totals ADD CONSTRAINT positive_destroyed_pool CHECK (destroyed_pool >= 0);
ALTER TABLE issuance_totals ADD CONSTRAINT positive_destroyed_confirmed CHECK (destroyed_confirmed >= 0);
