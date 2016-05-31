ALTER TABLE historical_account_outputs RENAME TO historical_outputs;
ALTER TABLE historical_outputs ALTER account_id DROP NOT NULL;
DROP INDEX historical_account_outputs_account_id_idx;
CREATE INDEX historical_outputs_account_id_idx ON historical_outputs USING btree (account_id) WHERE account_id IS NOT NULL;
ALTER INDEX historical_account_outputs_asset_id_idx RENAME TO historical_outputs_asset_id;
ALTER INDEX historical_account_outputs_timespan_idx RENAME TO historical_outputs_timespan_idx;
