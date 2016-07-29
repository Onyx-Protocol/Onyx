CREATE TABLE signers (
  id text NOT NULL,
  type text NOT NULL,
  key_index bigserial NOT NULL,
  xpubs text[] NOT NULL,
  quorum int NOT NULL,
  client_token text,
  archived boolean NOT NULL default 'false',
  PRIMARY KEY(id),
  UNIQUE(client_token)
);
CREATE INDEX ON signers (type, id);

CREATE TABLE account_control_programs (
  id text NOT NULL DEFAULT next_chain_id('acp'),
  signer_id text NOT NULL,
  key_index bigint NOT NULL,
  control_program bytea NOT NULL,
  redeem_program bytea NOT NULL
);
CREATE INDEX ON account_control_programs (control_program);

CREATE SEQUENCE account_control_program_seq
  START WITH 10001
  INCREMENT BY 10000
  NO MINVALUE
  NO MAXVALUE
  CACHE 1;

ALTER TABLE account_utxos DROP COLUMN manager_node_id;
ALTER TABLE account_utxos RENAME COLUMN addr_index TO control_program_index;
ALTER TABLE account_utxos RENAME COLUMN script TO control_program;

DROP TABLE manager_nodes, accounts, addresses, rotations;
DROP SEQUENCE address_index_seq;
