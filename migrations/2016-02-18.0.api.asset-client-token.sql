ALTER TABLE assets
ADD COLUMN client_token text,
ADD UNIQUE (issuer_node_id, client_token);
