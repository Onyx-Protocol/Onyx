CREATE TABLE applications (
    id text DEFAULT next_chain_id('app'::text) NOT NULL PRIMARY KEY,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE members (
    application_id text NOT NULL REFERENCES applications (id),
    user_id text NOT NULL REFERENCES users (id),
    role text NOT NULL CHECK (role = 'developer' OR role = 'admin'),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    UNIQUE(application_id, user_id)
);

CREATE INDEX ON members (user_id);
