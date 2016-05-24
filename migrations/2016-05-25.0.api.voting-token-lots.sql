ALTER TABLE voting_tokens
    DROP CONSTRAINT voting_tokens_pkey,
    ADD PRIMARY KEY (tx_hash, index);
