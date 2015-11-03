CREATE TABLE blocks (
	block_hash text PRIMARY KEY,
	height bigint NOT NULL UNIQUE,
	data bytea NOT NULL
);
