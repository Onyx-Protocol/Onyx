ALTER TABLE addresses DROP column address;
CREATE UNIQUE INDEX ON addresses (pk_script);
