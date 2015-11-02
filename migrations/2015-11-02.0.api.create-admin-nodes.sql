CREATE TABLE admin_nodes(
    id text DEFAULT next_chain_id('an'::text) NOT NULL,
    project_id text NOT NULL,
    label text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
