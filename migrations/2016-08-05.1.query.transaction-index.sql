CREATE TABLE annotated_txs (
    block_height  bigint    NOT NULL,
    tx_pos        integer   NOT NULL,
    tx_hash       text      NOT NULL,
    data          jsonb     NOT NULL,
    PRIMARY KEY(block_height, tx_pos)
);

CREATE INDEX annotated_txs_data ON annotated_txs USING GIN (data);
