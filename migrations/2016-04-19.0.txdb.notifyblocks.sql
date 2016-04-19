CREATE RULE block_notify AS
    ON INSERT TO blocks DO SELECT pg_notify('newblock', NEW.height::text);
