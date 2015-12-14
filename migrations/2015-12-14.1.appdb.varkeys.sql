ALTER TABLE issuer_nodes ADD COLUMN variable_keys integer DEFAULT 0 NOT NULL;
ALTER TABLE manager_nodes ALTER variable_keys SET NOT NULL;

