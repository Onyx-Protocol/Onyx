CREATE TABLE voting_tokens (
    asset_id        text     NOT NULL,
    right_asset_id  text     NOT NULL,
    tx_hash         text     NOT NULL,
    index           integer  NOT NULL,
    state           smallint NOT NULL,
    closed          boolean  NOT NULL,
    vote            smallint NOT NULL,
    option_count    integer  NOT NULL,
    secret_hash     text     NOT NULL,
    admin_script    bytea    NOT NULL,
    amount          bigint   NOT NULL,
    PRIMARY KEY(asset_id, right_asset_id)
);
