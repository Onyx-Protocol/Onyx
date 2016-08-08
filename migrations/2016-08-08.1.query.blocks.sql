CREATE TABLE query_blocks (
    height    bigint NOT NULL PRIMARY KEY,
    timestamp bigint NOT NULL
);

CREATE INDEX ON query_blocks (timestamp);
