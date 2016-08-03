ALTER TABLE issuer_nodes DROP COLUMN project_id;
ALTER TABLE manager_nodes DROP COLUMN project_id;
CREATE UNIQUE INDEX ON issuer_nodes (client_token);
CREATE UNIQUE INDEX ON manager_nodes (client_token);
DROP TABLE projects;
