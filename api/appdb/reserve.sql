CREATE OR REPLACE FUNCTION reserve_utxos(asset_id text, bucket_id text, amt bigint)
	RETURNS TABLE(txid text, index integer, amount bigint, address_id text)
	LANGUAGE plv8
	AS $$

	var q = plv8.prepare(
		"	WITH reserved AS ("+
		"		SELECT txid, index, amount, address_id FROM utxos"+
		"		WHERE asset_id=$1 AND bucket_id=$2"+
		"		AND reserved_at < now() - '60s'::interval"+
		"		ORDER BY address_id, txid, index ASC"+
		"		LIMIT 1"+
		"		FOR UPDATE"+
		"	)"+
		"	UPDATE utxos SET reserved_at=now() FROM reserved"+
		"	WHERE reserved.txid=utxos.txid AND reserved.index=utxos.index"+
		"	RETURNING reserved.txid, reserved.index, reserved.amount, reserved.address_id"
	);

	var selectedUTXOs = [];
	while(amt > 0) {
		var rows = q.execute([asset_id, bucket_id]);
		if (rows.length === 0) {
			throw new Error("insufficient funds");
		}
		amt -= rows[0]["amount"];
		selectedUTXOs.push(rows[0]);
	}
	return selectedUTXOs;
$$;

CREATE OR REPLACE FUNCTION reserve_tx_utxos(asset_id text, bucket_id text, txid text, amt bigint)
	RETURNS TABLE(txid text, index integer, amount bigint, address_id text)
	LANGUAGE plv8
	AS $$

	var q = plv8.prepare(
		"	WITH reserved AS ("+
		"		SELECT txid, index, amount, address_id FROM utxos"+
		"		WHERE asset_id=$1 AND bucket_id=$2 AND txid=$3"+
		"		AND reserved_at < now() - '60s'::interval"+
		"		ORDER BY address_id, txid, index ASC"+
		"		LIMIT 1"+
		"		FOR UPDATE"+
		"	)"+
		"	UPDATE utxos SET reserved_at=now() FROM reserved"+
		"	WHERE reserved.txid=utxos.txid AND reserved.index=utxos.index"+
		"	RETURNING reserved.txid, reserved.index, reserved.amount, reserved.address_id"
	);

	var selectedUTXOs = [];
	while(amt > 0) {
		var rows = q.execute([asset_id, bucket_id, txid]);
		if (rows.length === 0) {
			throw new Error("insufficient funds");
		}
		amt -= rows[0]["amount"];
		selectedUTXOs.push(rows[0]);
	}
	return selectedUTXOs;
$$;
