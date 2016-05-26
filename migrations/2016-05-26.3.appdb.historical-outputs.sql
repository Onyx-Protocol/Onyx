CREATE TABLE historical_account_outputs (
  tx_hash    text      NOT NULL,
  index      integer   NOT NULL,
  asset_id   text      NOT NULL,
  amount     bigint    NOT NULL,
  account_id text      NOT NULL,
  timespan   int8range NOT NULL
);

CREATE INDEX historical_account_outputs_timespan_idx ON historical_account_outputs USING gist (timespan);
CREATE INDEX historical_account_outputs_asset_id_idx ON historical_account_outputs USING btree (asset_id);
CREATE INDEX historical_account_outputs_account_id_idx ON historical_account_outputs USING btree (account_id);