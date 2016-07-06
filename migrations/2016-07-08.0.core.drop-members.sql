ALTER TABLE invitations
	DROP CONSTRAINT invitations_project_id_fkey,
	DROP COLUMN project_id;

ALTER TABLE users
	ADD COLUMN role text NOT NULL DEFAULT 'developer',
	ADD CONSTRAINT users_role_check CHECK (((role = 'developer'::text) OR (role = 'admin'::text)));

WITH admin_ids AS (
	SELECT DISTINCT user_id FROM members WHERE role = 'admin'
)
UPDATE users
SET role = 'admin'
WHERE id IN (SELECT user_id FROM admin_ids);

DROP TABLE members;
