CREATE TABLE submitted_txs (
    tx_id        text      NOT NULL PRIMARY KEY,
    height       bigint    NOT NULL,
    submitted_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);
