CREATE OR REPLACE FUNCTION reserve_utxos(asset_id text, bucket_id text, amt bigint, ttl interval)
	RETURNS TABLE(txid text, index integer, amount bigint, address_id text)
	LANGUAGE plv8
	AS $$

	var countQ = plv8.prepare(
		"SELECT COUNT(*) AS cnt FROM utxos "+
		"WHERE asset_id=$1 AND bucket_id=$2 "+
		"AND reserved_until < now()"
	)
	var count = parseInt(countQ.execute([asset_id, bucket_id])[0]["cnt"]);
	var max = 10;

	var q = plv8.prepare(
		"	WITH reserved AS ("+
		"		SELECT txid, index, amount, address_id FROM utxos"+
		"		WHERE asset_id=$1 AND bucket_id=$2"+
		"		AND reserved_until < now()"+
		"		LIMIT 1"+
		"		OFFSET $3"+
		"		FOR UPDATE"+
		"	)"+
		"	UPDATE utxos SET reserved_until=now()+$4::interval FROM reserved"+
		"	WHERE reserved.txid=utxos.txid AND reserved.index=utxos.index"+
		"	RETURNING reserved.txid, reserved.index, reserved.amount, reserved.address_id"
	);

	var selectedUTXOs = [];
	while(amt > 0) {
		var off = Math.floor(Math.random() * Math.min(count, max));
		var rows = q.execute([asset_id, bucket_id, off, ttl]);
		if (rows.length === 0) {
			throw new Error("insufficient funds");
		}
		amt -= rows[0]["amount"];
		selectedUTXOs.push(rows[0]);

		if(count > 0) count--;
	}
	return selectedUTXOs;
$$;

CREATE OR REPLACE FUNCTION reserve_tx_utxos(asset_id text, bucket_id text, txid text, amt bigint, ttl interval)
	RETURNS TABLE(txid text, index integer, amount bigint, address_id text)
	LANGUAGE plv8
	AS $$

	var q = plv8.prepare(
		"	WITH reserved AS ("+
		"		SELECT txid, index, amount, address_id FROM utxos"+
		"		WHERE asset_id=$1 AND bucket_id=$2 AND txid=$3"+
		"		AND reserved_until < now()"+
		"		ORDER BY address_id, txid, index ASC"+
		"		LIMIT 1"+
		"		FOR UPDATE"+
		"	)"+
		"	UPDATE utxos SET reserved_until=now()+$4::interval FROM reserved"+
		"	WHERE reserved.txid=utxos.txid AND reserved.index=utxos.index"+
		"	RETURNING reserved.txid, reserved.index, reserved.amount, reserved.address_id"
	);

	var selectedUTXOs = [];
	while(amt > 0) {
		var rows = q.execute([asset_id, bucket_id, txid, ttl]);
		if (rows.length === 0) {
			throw new Error("insufficient funds");
		}
		amt -= rows[0]["amount"];
		selectedUTXOs.push(rows[0]);
	}
	return selectedUTXOs;
$$;
