CREATE TABLE manager_txs (
    id text DEFAULT next_chain_id('mtx'::text) NOT NULL,
    manager_node_id text NOT NULL,
    data json NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    txid text NOT NULL
);

CREATE TABLE manager_txs_accounts (
    manager_tx_id text NOT NULL,
    account_id text NOT NULL
);

CREATE TABLE issuer_txs (
    id text DEFAULT next_chain_id('itx'::text) NOT NULL,
    issuer_node_id text NOT NULL,
    data json NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    txid text NOT NULL
);

CREATE TABLE issuer_txs_assets (
    issuer_tx_id text NOT NULL,
    asset_id text NOT NULL
);
