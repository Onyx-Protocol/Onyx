CREATE TABLE auth_tokens (
    id text DEFAULT next_chain_id('at'::text) NOT NULL,
    secret_hash bytea NOT NULL,
    user_id text REFERENCES users (id),
    type text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone
);
