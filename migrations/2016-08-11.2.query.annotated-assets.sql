CREATE TABLE annotated_assets (
    id    text NOT NULL PRIMARY KEY,
    data jsonb NOT NULL
);

CREATE INDEX annotated_assets_jsondata_idx ON annotated_assets USING gin (data jsonb_path_ops);
