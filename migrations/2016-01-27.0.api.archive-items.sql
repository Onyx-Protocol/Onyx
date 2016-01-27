ALTER TABLE assets ADD COLUMN archived boolean NOT NULL default false;
ALTER TABLE accounts ADD COLUMN archived boolean NOT NULL default false;
ALTER TABLE issuer_nodes ADD COLUMN archived boolean NOT NULL default false;
ALTER TABLE manager_nodes ADD COLUMN archived boolean NOT NULL default false;
ALTER TABLE projects ADD COLUMN archived boolean NOT NULL default false;
