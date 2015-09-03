CREATE TABLE users (
    id text NOT NULL PRIMARY KEY DEFAULT next_chain_id('u'::text),
    email text NOT NULL,
    password_hash bytea NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX on users (lower(email));
