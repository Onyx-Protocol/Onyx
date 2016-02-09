CREATE TABLE IF NOT EXISTS migrations (
  filename text NOT NULL,
  hash text NOT NULL,
  applied_at timestamp with time zone DEFAULT now() NOT NULL,
  PRIMARY KEY(filename)
);

INSERT INTO migrations (filename, hash, applied_at) VALUES('select.sql', 'b4e0497804e46e0a0b0b8c31975b062152d551bac49c3c2e80932567b4085dcd', '2016-02-09T23:21:55 US/Pacific');
