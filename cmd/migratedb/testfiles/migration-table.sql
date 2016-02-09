CREATE TABLE IF NOT EXISTS migrations (
  filename text NOT NULL,
  hash text NOT NULL,
  applied_at timestamp with time zone DEFAULT now() NOT NULL,
  PRIMARY KEY(filename)
);
