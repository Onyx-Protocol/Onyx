ALTER TABLE manager_nodes
    ADD COLUMN client_token text,
    ADD UNIQUE (project_id, client_token);
ALTER TABLE issuer_nodes
    ADD COLUMN client_token text,
    ADD UNIQUE (project_id, client_token);
