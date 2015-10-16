ALTER TABLE ONLY rotations DROP CONSTRAINT rotations_manager_node_id_fkey;
ALTER TABLE ONLY rotations ADD CONSTRAINT rotations_manager_node_id_fkey FOREIGN KEY (manager_node_id) REFERENCES manager_nodes(id) ON DELETE CASCADE;
