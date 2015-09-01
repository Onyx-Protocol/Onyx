ALTER TABLE wallets
	DROP chain_keys,
	DROP development,
	ALTER block_chain SET NOT NULL,
	ALTER block_chain SET DEFAULT 'sandbox',
	ALTER sigs_required SET DEFAULT 1,
	ALTER current_rotation DROP NOT NULL,
	DROP pek;

ALTER TABLE keys
	DROP type,
	DROP enc_xpriv;

ALTER TABLE rotations
	DROP pek_pub;
