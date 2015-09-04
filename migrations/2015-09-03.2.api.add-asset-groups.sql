CREATE TABLE asset_groups (
    id text DEFAULT next_chain_id('ag'::text) NOT NULL PRIMARY KEY,
    application_id text NOT NULL,
    block_chain text DEFAULT 'sandbox'::text NOT NULL,
    sigs_required integer DEFAULT 1 NOT NULL,
    key_index bigserial NOT NULL,
    label text NOT NULL,
    keyset text[] NOT NULL,
    next_asset_index bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX asset_groups_application_id_idx ON asset_groups USING btree (application_id);
