CREATE TABLE issuance_activity (
    id text DEFAULT next_chain_id('iact'::text) NOT NULL PRIMARY KEY,
    asset_group_id text NOT NULL REFERENCES asset_groups (id),
    data json NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    txid text NOT NULL
);

CREATE UNIQUE INDEX ON issuance_activity USING btree (asset_group_id, txid);
CREATE INDEX ON issuance_activity (asset_group_id);

CREATE TABLE issuance_activity_assets (
    issuance_activity_id text NOT NULL REFERENCES issuance_activity (id),
    asset_id text NOT NULL REFERENCES assets (id)
);

CREATE UNIQUE INDEX ON issuance_activity_assets USING btree (issuance_activity_id, asset_id);
CREATE INDEX ON issuance_activity_assets (asset_id);
