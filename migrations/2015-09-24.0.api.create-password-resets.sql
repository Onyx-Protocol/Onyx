ALTER TABLE users
	ADD COLUMN pwreset_secret_hash bytea,
	ADD COLUMN pwreset_expires_at timestamp with time zone;
