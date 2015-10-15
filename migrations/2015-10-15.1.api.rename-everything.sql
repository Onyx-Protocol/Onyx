ALTER TABLE applications RENAME TO projects;
ALTER TABLE projects ALTER id SET DEFAULT next_chain_id('proj'::text);
ALTER TABLE projects RENAME CONSTRAINT applications_pkey TO projects_pkey;

ALTER TABLE wallets RENAME TO manager_nodes;
ALTER TABLE manager_nodes ALTER id SET DEFAULT next_chain_id('mn'::text);
CREATE SEQUENCE manager_nodes_key_index_seq
  START WITH 1
  INCREMENT BY 1
  NO MINVALUE
  NO MAXVALUE
  CACHE 1;
SELECT setval('manager_nodes_key_index_seq', (SELECT nextval('wallets_key_index_seq')+1000), false);
DROP SEQUENCE wallets_key_index_seq CASCADE;

ALTER TABLE asset_groups RENAME TO issuer_nodes;
ALTER TABLE issuer_nodes ALTER id SET DEFAULT next_chain_id('in'::text);
CREATE SEQUENCE issuer_nodes_key_index_seq
  START WITH 1
  INCREMENT BY 1
  NO MINVALUE
  NO MAXVALUE
  CACHE 1;
SELECT setval('issuer_nodes_key_index_seq', (SELECT nextval('asset_groups_key_index_seq')+1000), false);
DROP SEQUENCE asset_groups_key_index_seq CASCADE;

ALTER TABLE buckets RENAME TO accounts;
ALTER TABLE accounts ALTER id SET DEFAULT next_chain_id('acc'::text);
ALTER TABLE activity_buckets RENAME TO activity_accounts;

ALTER TABLE accounts RENAME wallet_id TO manager_node_id;
ALTER INDEX buckets_pkey RENAME TO accounts_pkey;
ALTER INDEX buckets_wallet_path RENAME TO accounts_manager_node_path;
ALTER TABLE accounts RENAME CONSTRAINT buckets_wallet_id_fkey TO accounts_manager_node_id_fkey;

ALTER TABLE activity RENAME wallet_id TO manager_node_id;
ALTER INDEX activity_wallet_id_txid_idx RENAME TO activity_manager_node_id_txid_idx;
ALTER INDEX activity_wallet_id_idx RENAME TO activity_manager_node_id_idx;
ALTER TABLE activity RENAME CONSTRAINT activity_wallet_id_fkey TO activity_manager_node_id_fkey;

ALTER TABLE activity_accounts RENAME bucket_id TO account_id;
ALTER INDEX activity_buckets_activity_id_bucket_id_idx RENAME TO activity_accounts_activity_id_account_id_idx;
ALTER INDEX activity_buckets_bucket_id_idx RENAME TO activity_accounts_account_id_idx;
ALTER TABLE activity_accounts RENAME CONSTRAINT activity_buckets_activity_id_fkey TO activity_accounts_activity_id_fkey;
ALTER TABLE activity_accounts RENAME CONSTRAINT activity_buckets_bucket_id_fkey TO activity_accounts_account_id_fkey;

ALTER TABLE addresses RENAME wallet_id TO manager_node_id;
ALTER TABLE addresses RENAME bucket_id TO account_id;
ALTER INDEX addresses_bucket_id_key_index_idx RENAME TO addresses_account_id_key_index_idx;
ALTER INDEX addresses_bucket_id_idx RENAME TO addresses_account_id_idx;
ALTER INDEX addresses_wallet_id_idx RENAME TO addresses_manager_node_id_idx;
ALTER TABLE addresses RENAME CONSTRAINT addresses_bucket_id_fkey TO addresses_account_id_fkey;
ALTER TABLE addresses RENAME CONSTRAINT addresses_wallet_id_fkey TO addresses_manager_node_id_fkey;

ALTER TABLE issuance_activity RENAME asset_group_id TO issuer_node_id;
ALTER INDEX issuance_activity_asset_group_id_idx RENAME TO issuance_activity_issuer_node_id_idx;
ALTER INDEX issuance_activity_asset_group_id_txid_idx RENAME TO issuance_activity_issuer_node_id_txid_idx;
ALTER TABLE issuance_activity RENAME CONSTRAINT issuance_activity_asset_group_id_fkey TO issuance_activity_issuer_node_id_fkey;

ALTER TABLE issuer_nodes RENAME application_id TO project_id;
ALTER INDEX asset_groups_pkey RENAME TO issuer_nodes_pkey;
ALTER INDEX asset_groups_application_id_idx RENAME TO issuer_nodes_project_id_idx;

ALTER TABLE manager_nodes RENAME application_id TO project_id;
ALTER TABLE manager_nodes RENAME next_bucket_index TO next_account_index;
ALTER TABLE manager_nodes RENAME buckets_count TO accounts_count;
ALTER INDEX wallets_pkey RENAME TO manager_nodes_pkey;
ALTER INDEX wallets_application_id_idx RENAME TO manager_nodes_project_id_idx;
ALTER TABLE manager_nodes RENAME CONSTRAINT wallets_application_id_fkey TO manager_nodes_project_id_fkey;

ALTER TABLE assets RENAME asset_group_id TO issuer_node_id;
ALTER INDEX assets_asset_group_id_idx RENAME TO assets_issuer_node_id_idx;
ALTER TABLE assets RENAME CONSTRAINT assets_asset_group_id_fkey TO assets_issuer_node_id_fkey;

ALTER TABLE invitations RENAME application_id TO project_id;
ALTER TABLE invitations RENAME CONSTRAINT invitations_application_id_fkey TO invitations_project_id_fkey;

ALTER TABLE members RENAME application_id TO project_id;
ALTER INDEX members_application_id_user_id_key RENAME TO members_project_id_user_id_key;
ALTER TABLE members RENAME CONSTRAINT members_application_id_fkey TO members_project_id_fkey;

ALTER TABLE rotations RENAME wallet_id TO manager_node_id;
ALTER TABLE rotations RENAME CONSTRAINT rotations_wallet_id_fkey TO rotations_manager_node_id_fkey;

ALTER TABLE utxos RENAME bucket_id TO account_id;
ALTER TABLE utxos RENAME wallet_id TO manager_node_id;
ALTER INDEX utxos_bucket_id_asset_id_reserved_at_idx RENAME TO utxos_account_id_asset_id_reserved_at_idx;
ALTER INDEX utxos_wallet_id_asset_id_reserved_at_idx RENAME TO utxos_manager_node_id_asset_id_reserved_at_idx;

ALTER SEQUENCE manager_nodes_key_index_seq OWNED BY manager_nodes.key_index;
ALTER SEQUENCE issuer_nodes_key_index_seq OWNED BY issuer_nodes.key_index;
ALTER TABLE ONLY manager_nodes ALTER COLUMN key_index SET DEFAULT nextval('manager_nodes_key_index_seq'::regclass);
ALTER TABLE ONLY issuer_nodes ALTER COLUMN key_index SET DEFAULT nextval('issuer_nodes_key_index_seq'::regclass);
