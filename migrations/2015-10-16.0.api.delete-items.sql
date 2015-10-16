ALTER TABLE ONLY accounts DROP CONSTRAINT accounts_manager_node_id_fkey;
ALTER TABLE ONLY accounts ADD CONSTRAINT accounts_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id) ON DELETE NO ACTION;

ALTER TABLE ONLY activity_accounts DROP CONSTRAINT activity_accounts_account_id_fkey;
ALTER TABLE ONLY activity_accounts ADD CONSTRAINT activity_accounts_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE NO ACTION;

ALTER TABLE ONLY activity DROP CONSTRAINT activity_manager_node_id_fkey;
ALTER TABLE ONLY activity ADD CONSTRAINT activity_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id) ON DELETE NO ACTION;

ALTER TABLE ONLY addresses DROP CONSTRAINT addresses_account_id_fkey;
ALTER TABLE ONLY addresses ADD CONSTRAINT addresses_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE NO ACTION;

ALTER TABLE ONLY addresses DROP CONSTRAINT addresses_manager_node_id_fkey;
ALTER TABLE ONLY addresses ADD CONSTRAINT addresses_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id) ON DELETE NO ACTION;

ALTER TABLE ONLY assets DROP CONSTRAINT assets_issuer_node_id_fkey;
ALTER TABLE ONLY assets ADD CONSTRAINT assets_issuer_node_id_fkey FOREIGN KEY (issuer_node_id) REFERENCES issuer_nodes(id) ON DELETE NO ACTION;

ALTER TABLE ONLY issuance_activity_assets DROP CONSTRAINT issuance_activity_assets_asset_id_fkey;
ALTER TABLE ONLY issuance_activity_assets ADD CONSTRAINT issuance_activity_assets_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE NO ACTION;

ALTER TABLE ONLY issuance_activity DROP CONSTRAINT issuance_activity_issuer_node_id_fkey;
ALTER TABLE ONLY issuance_activity ADD CONSTRAINT issuance_activity_issuer_node_id_fkey FOREIGN KEY (issuer_node_id) REFERENCES issuer_nodes(id) ON DELETE NO ACTION;

ALTER TABLE ONLY rotations DROP CONSTRAINT rotations_manager_node_id_fkey;
ALTER TABLE ONLY rotations ADD CONSTRAINT rotations_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id) ON DELETE NO ACTION;
