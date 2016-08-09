CREATE TABLE annotated_outputs (
	block_height bigint    NOT NULL,
	tx_pos       int       NOT NULL,
	output_index int       NOT NULL,
	tx_hash      text      NOT NULL,
	data         jsonb     NOT NULL,
	timespan     int8range NOT NULL,

	PRIMARY KEY (block_height, tx_pos, output_index)
);

CREATE INDEX annotated_outputs_timespan_idx ON annotated_outputs USING gist (timespan);
CREATE INDEX annotated_outputs_jsondata_idx ON annotated_outputs USING gin (data jsonb_path_ops);
CREATE INDEX annotated_outputs_outpoint_idx ON annotated_outputs (tx_hash, output_index);
