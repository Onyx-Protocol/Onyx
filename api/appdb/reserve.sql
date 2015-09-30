CREATE OR REPLACE FUNCTION reserve_utxos(asset_id text, bucket_id text, amt bigint, ttl interval)
	RETURNS TABLE(txid text, index integer, amount bigint, address_id text)
	LANGUAGE plv8
	AS $$

	var candidateRowQ = plv8.prepare(
		"	SELECT txid, index, amount FROM utxos "+
		"	WHERE asset_id=$1 AND bucket_id=$2 "+
		"		AND reserved_until < now()"
	);
	var candidateRows = candidateRowQ.execute([asset_id, bucket_id]);
	var txids = [];
	var indexes = [];
	var amountNeeded = amt;
	for (var i = 0; i < candidateRows.length; ++i) {
		txids.push(candidateRows[i]['txid']);
		indexes.push(candidateRows[i]['index']);
		amountNeeded -= candidateRows[i]['amount'];
		if (amountNeeded <= 0) {
			 break;
		}
	}
	if (amountNeeded > 0) {
		throw new Error("insufficient funds");
	}

	var lockQ = plv8.prepare(
		"	WITH outpoints AS ("+
		"		SELECT unnest($1::text[]), unnest($2::int[])"+
		"	), locked AS ("+
		"		SELECT txid, index FROM utxos"+
		"		WHERE reserved_until < now() AND (txid, index) IN (TABLE outpoints)"+
		"		FOR UPDATE NOWAIT"+
		"	)"+
		"	SELECT COUNT(*) AS cnt FROM locked;"
	);
	var rows = lockQ.execute([txids, indexes]);
	if (parseInt(rows[0]['cnt']) != txids.length) {
		throw new Error("candidate rows deleted before lock");
	}

	var updateQ = plv8.prepare(
		"	WITH outpoints AS ("+
		"		SELECT unnest($1::text[]), unnest($2::int[])"+
		"	)"+
		"	UPDATE utxos SET reserved_until=now()+$3::interval"+
		"	WHERE (txid, index) IN (TABLE outpoints)"+
		"	RETURNING txid, index, amount, address_id;"
	);

	return updateQ.execute([txids, indexes, ttl]);
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
