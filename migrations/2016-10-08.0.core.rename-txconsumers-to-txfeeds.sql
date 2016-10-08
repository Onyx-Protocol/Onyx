ALTER TABLE txconsumers RENAME TO txfeeds;
ALTER TABLE ONLY txfeeds
    ADD CONSTRAINT txfeeds_alias_key UNIQUE (alias);
ALTER TABLE ONLY txfeeds
    DROP CONSTRAINT txconsumers_alias_key;
ALTER TABLE ONLY txfeeds
    ADD CONSTRAINT txfeeds_client_token_key UNIQUE (client_token);
ALTER TABLE ONLY txfeeds
    DROP CONSTRAINT txconsumers_client_token_key;
ALTER INDEX txconsumers_pkey RENAME TO txfeeds_pkey;
