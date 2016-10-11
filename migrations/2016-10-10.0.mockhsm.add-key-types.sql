ALTER TABLE mockhsm RENAME COLUMN xpub TO pub;
ALTER TABLE mockhsm RENAME COLUMN xprv TO prv;
ALTER TABLE mockhsm
  DROP COLUMN xpub_hash,
  ADD COLUMN key_type text NOT NULL DEFAULT 'chain_kd';