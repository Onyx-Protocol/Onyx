ALTER TABLE mockhsm ADD COLUMN sort_id bigint;

CREATE SEQUENCE mockhsm_sort_id_seq
  START WITH 1
  INCREMENT BY 1
  NO MINVALUE
  NO MAXVALUE
  CACHE 1;

UPDATE mockhsm
  SET sort_id = nextval('mockhsm_sort_id_seq'::regclass)
FROM (SELECT xpub FROM mockhsm ORDER BY alias, xpub) AS ordered
WHERE mockhsm.xpub = ordered.xpub;

ALTER TABLE mockhsm
  ALTER COLUMN sort_id SET DEFAULT nextval('mockhsm_sort_id_seq'::regclass),
  ALTER COLUMN sort_id SET NOT NULL,
  ADD CONSTRAINT sort_id_index UNIQUE (sort_id);
