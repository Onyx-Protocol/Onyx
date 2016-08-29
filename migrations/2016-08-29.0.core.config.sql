CREATE TABLE config (
  singleton boolean DEFAULT true NOT NULL PRIMARY KEY,
  is_signer boolean,
  is_generator boolean,
  genesis_hash text NOT NULL,
  remote_generator_url text,
  configured_at timestamptz NOT NULL,
  CONSTRAINT config_singleton CHECK (singleton)
);