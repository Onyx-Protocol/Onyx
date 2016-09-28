CREATE TYPE access_token_type AS ENUM('client', 'network');
CREATE TABLE access_tokens (
	id text NOT NULL,
	sort_id text DEFAULT next_chain_id('at'),
	type access_token_type NOT NULL,
	hashed_secret bytea NOT NULL,
	created timestamptz NOT NULL DEFAULT NOW(),
	PRIMARY KEY(id)
);
