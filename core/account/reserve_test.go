package account

import (
	"context"
	"testing"
	"time"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
)

const sampleAccountUTXOs = `
	INSERT INTO account_utxos
	(output_id, asset_id, amount, account_id, control_program_index,
     control_program, confirmed_in, source_id, source_pos, ref_data_hash, change) VALUES (
		decode('9886ae2dc24b6d868c68768038c43801e905a62f1a9b826ca0dc357f00c30117', 'hex'),
		decode('df1df9d4f66437ab5be715e4d1faeb29d24c80a6dc8276d6a630f05c5f1f7693', 'hex'),
		1000, 'accEXAMPLE', 1, '\x6a'::bytea, 1,
		decode('905a62f1a9b826ca0dc357f00c301179886ae2dc24b6d868c68768038c43801e', 'hex'),
		0, decode('0000000000000000000000000000000000000000000000000000000000000000', 'hex'),
		false);
`

func TestCancelReservation(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)
	_, err := db.ExecContext(ctx, sampleAccountUTXOs)
	if err != nil {
		t.Fatal(err)
	}

	// Create a Chain with our output already in the state tree.
	var outid bc.Hash
	err = outid.UnmarshalText([]byte("9886ae2dc24b6d868c68768038c43801e905a62f1a9b826ca0dc357f00c30117"))
	if err != nil {
		t.Fatal(err)
	}
	c := prottest.NewChain(t, prottest.WithOutputIDs(outid))

	utxoDB := newReserver(db, c, nil)
	res, err := utxoDB.ReserveUTXO(ctx, outid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the UTXO is reserved.
	_, err = utxoDB.ReserveUTXO(ctx, outid, nil, time.Now())
	if err != ErrReserved {
		t.Fatalf("got=%s want=%s", err, ErrReserved)
	}

	// Cancel the reservation.
	err = utxoDB.Cancel(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Reserving again should succeed.
	_, err = utxoDB.ReserveUTXO(ctx, outid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}
