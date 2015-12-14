ALTER TABLE manager_nodes ALTER variable_keys SET DEFAULT 0;
UPDATE manager_nodes SET variable_keys = 0 WHERE variable_keys IS NULL;
