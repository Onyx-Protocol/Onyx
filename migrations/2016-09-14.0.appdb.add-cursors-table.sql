CREATE TABLE cursors (
  id text PRIMARY KEY,
  alias text UNIQUE,
  filter text,
  after text,
  is_ascending boolean,
  client_token text NOT NULL UNIQUE
);