ALTER TABLE accounts
ADD COLUMN client_token text,
ADD UNIQUE (manager_node_id, client_token);
