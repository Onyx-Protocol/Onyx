ALTER TABLE assets RENAME issued TO issued_pool;
ALTER TABLE assets ADD COLUMN issued_confirmed bigint DEFAULT 0 NOT NULL;
ALTER TABLE assets ADD CONSTRAINT positive_issued_pool CHECK (issued_pool >= 0);
ALTER TABLE assets ADD CONSTRAINT positive_issued_confirmed CHECK (issued_confirmed >= 0);

