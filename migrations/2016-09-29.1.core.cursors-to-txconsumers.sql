ALTER TABLE cursors RENAME TO txconsumers;
ALTER TABLE ONLY txconsumers
    ADD CONSTRAINT txconsumers_alias_key UNIQUE (alias);
ALTER TABLE ONLY txconsumers
    DROP CONSTRAINT cursors_alias_key;
ALTER TABLE ONLY txconsumers
    ADD CONSTRAINT txconsumers_client_token_key UNIQUE (client_token);
ALTER TABLE ONLY txconsumers
    DROP CONSTRAINT cursors_client_token_key;
ALTER INDEX cursors_pkey RENAME TO txconsumers_pkey;
