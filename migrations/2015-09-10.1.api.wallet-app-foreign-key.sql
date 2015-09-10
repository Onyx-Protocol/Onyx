ALTER TABLE ONLY wallets
    ADD CONSTRAINT wallets_application_id_fkey FOREIGN KEY (application_id) REFERENCES applications(id);
