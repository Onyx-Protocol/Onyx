CREATE TABLE voting_right_txs (
    tx_hash text NOT NULL,
    index integer NOT NULL,
    asset_id text NOT NULL,
    account_id text,
    holder bytea NOT NULL,
    deadline bigint,
    delegatable boolean NOT NULL,
    ownership_chain bytea NOT NULL,
    PRIMARY KEY(tx_hash, index)
);
