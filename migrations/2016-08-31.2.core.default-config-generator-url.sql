ALTER TABLE config ADD COLUMN generator_url text NOT NULL DEFAULT '';
UPDATE config SET generator_url = remote_generator_url WHERE remote_generator_url IS NOT NULL;
ALTER TABLE config DROP COLUMN remote_generator_url;
