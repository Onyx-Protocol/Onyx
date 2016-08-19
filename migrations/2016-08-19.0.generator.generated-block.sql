CREATE TABLE generator_pending_block (
    singleton boolean DEFAULT true NOT NULL PRIMARY KEY,
    data      bytea   NOT NULL,
    CONSTRAINT generator_pending_block_singleton CHECK (singleton)
);
