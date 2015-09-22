CREATE TABLE invitations (
    id text NOT NULL PRIMARY KEY, -- We'll use something less guessable than next_chain_id
    application_id text NOT NULL REFERENCES applications(id),
    email text NOT NULL,
    role text NOT NULL CHECK (role = 'developer' OR role = 'admin'),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
