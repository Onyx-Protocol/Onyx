CREATE TABLE leader (
    singleton boolean DEFAULT true NOT NULL,
    leader_key text NOT NULL,
    expiry timestamp with time zone DEFAULT '1970-01-01 00:00:00-08'::timestamp with time zone NOT NULL,
    CONSTRAINT leader_singleton CHECK (singleton)
);

ALTER TABLE ONLY leader
    ADD CONSTRAINT leader_singleton_key UNIQUE (singleton);
