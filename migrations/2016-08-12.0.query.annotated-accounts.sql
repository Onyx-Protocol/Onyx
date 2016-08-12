CREATE TABLE annotated_accounts (
    id     text  NOT NULL PRIMARY KEY,
    data   jsonb NOT NULL
);
CREATE INDEX annotated_accounts_jsondata_idx ON annotated_accounts USING gin (data jsonb_path_ops);
