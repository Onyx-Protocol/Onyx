package wallet

//go:generate go run gen.go wallet reserveSQL reserve.sql
const reserveSQL = `CREATE OR REPLACE FUNCTION reserve_outputs(asset_id text, bucket_id text, amt bigint)
	RETURNS TABLE(txid text, index integer, amount bigint)
	LANGUAGE plv8
	AS $$

	var q = plv8.prepare(
		"	WITH reserved AS ("+
		"		SELECT txid, index, amount FROM outputs"+
		"		WHERE asset_id=$1 AND bucket_id=$2"+
		"		AND reserved_at < now() - '60s'::interval"+
		"		ORDER BY receiver_id, txid, index ASC"+
		"		LIMIT 1"+
		"	)"+
		"	UPDATE outputs SET reserved_at=NOW() FROM reserved"+
		"	WHERE reserved.txid=outputs.txid AND reserved.index=outputs.index"+
		"	RETURNING reserved.txid, reserved.index, reserved.amount"
	);
	var selectedOutputs = [];
	while(amt > 0) {
		var rows = q.execute([asset_id, bucket_id]);
		if (rows.length === 0) {
			return null; // insufficient funds
		}
		amt -= rows[0]["amount"];
		selectedOutputs.push(rows[0]);
	}
	return selectedOutputs;
$$;
`
