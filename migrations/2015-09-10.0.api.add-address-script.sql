ALTER TABLE addresses
	ALTER id SET DEFAULT next_chain_id('a'),
	ADD redeem_script bytea NOT NULL,
	ADD pk_script bytea NOT NULL;
