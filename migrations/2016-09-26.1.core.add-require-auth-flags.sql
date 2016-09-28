ALTER TABLE config
	ADD COLUMN network_authed bool DEFAULT 'f',
	ADD COLUMN client_authed bool DEFAULT 'f';
