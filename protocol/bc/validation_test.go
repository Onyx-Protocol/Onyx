package bc

import (
	"math"
	"testing"

	"chain/protocol/vm"

	"github.com/davecgh/go-spew/spew"
)

func TestTxValidation(t *testing.T) {
	var (
		tx *TxEntries
		vs *validationState

		// the mux from tx, pulled out for convenience
		mux *Mux
	)

	cases := []struct {
		desc string // description of the test case
		f    func() // function to adjust tx, vs, and/or mux
		err  error  // expected error
	}{
		{
			desc: "base case",
		},
		{
			desc: "failing mux program",
			f: func() {
				mux.Body.Program.Code = []byte{byte(vm.OP_FALSE)}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "unbalanced mux amounts",
			f: func() {
				mux.Body.Sources[0].Value.Amount++
				mux.Body.Sources[0].Entry.(*Issuance).Witness.Destination.Value.Amount++
			},
			err: errUnbalanced,
		},
		{
			desc: "overflowing mux source amounts",
			f: func() {
				mux.Body.Sources[0].Value.Amount = math.MaxInt64
				mux.Body.Sources[0].Entry.(*Issuance).Witness.Destination.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "underflowing mux destination amounts",
			f: func() {
				mux.Witness.Destinations[0].Value.Amount = math.MaxInt64
				mux.Witness.Destinations[0].Entry.(*Output).Body.Source.Value.Amount = math.MaxInt64
				mux.Witness.Destinations[1].Value.Amount = math.MaxInt64
				mux.Witness.Destinations[1].Entry.(*Output).Body.Source.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "unbalanced mux assets",
			f: func() {
				mux.Body.Sources[0].Value.AssetID = AssetID{255}
				mux.Body.Sources[0].Entry.(*Issuance).Witness.Destination.Value.AssetID = AssetID{255}
			},
			err: errUnbalanced,
		},
		{
			desc: "nonempty mux exthash",
			f: func() {
				mux.Body.ExtHash = Hash{1}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonempty mux exthash, but that's OK",
			f: func() {
				tx.Body.Version = 2
				mux.Body.ExtHash = Hash{1}
			},
		},
		{
			desc: "failing nonce program",
			f: func() {
				iss := tx.TxInputs[0].(*Issuance)
				nonce := iss.Anchor.(*Nonce)
				nonce.Body.Program.Code = []byte{byte(vm.OP_FALSE)}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "nonce exthash nonempty",
			f: func() {
				iss := tx.TxInputs[0].(*Issuance)
				nonce := iss.Anchor.(*Nonce)
				nonce.Body.ExtHash = Hash{1}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonce exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				iss := tx.TxInputs[0].(*Issuance)
				nonce := iss.Anchor.(*Nonce)
				nonce.Body.ExtHash = Hash{1}
			},
		},
		{
			desc: "nonce timerange misordered",
			f: func() {
				iss := tx.TxInputs[0].(*Issuance)
				nonce := iss.Anchor.(*Nonce)
				tr := nonce.TimeRange
				tr.Body.MinTimeMS = tr.Body.MaxTimeMS + 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "nonce timerange disagrees with tx timerange",
			f: func() {
				iss := tx.TxInputs[0].(*Issuance)
				nonce := iss.Anchor.(*Nonce)
				tr := nonce.TimeRange
				tr.Body.MaxTimeMS = tx.Body.MaxTimeMS - 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "nonce timerange exthash nonempty",
			f: func() {
				iss := tx.TxInputs[0].(*Issuance)
				nonce := iss.Anchor.(*Nonce)
				tr := nonce.TimeRange
				tr.Body.ExtHash = Hash{1}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonce timerange exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				iss := tx.TxInputs[0].(*Issuance)
				nonce := iss.Anchor.(*Nonce)
				tr := nonce.TimeRange
				tr.Body.ExtHash = Hash{1}
			},
		},
		{
			desc: "mismatched output source / mux dest position",
			f: func() {
				tx.Results[0].(*Output).Body.Source.Position = 1
			},
			err: errMismatchedPosition,
		},
		{
			desc: "mismatched output source and mux dest",
			f: func() {
				mux.Witness.Destinations[0].Ref = Hash{1}
			},
			err: errMismatchedReference,
		},
		{
			desc: "invalid mux destinaton position",
			f: func() {
				mux.Witness.Destinations[0].Position = 1
			},
			err: errPosition,
		},
		{
			desc: "mismatched mux dest value / output source value",
			f: func() {
				mux.Witness.Destinations[0].Value.Amount = tx.Results[0].(*Output).Body.Source.Value.Amount + 1
			},
			err: errMismatchedValue,
		},
		{
			desc: "output exthash nonempty",
			f: func() {
				tx.Results[0].(*Output).Body.ExtHash = Hash{1}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "output exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Results[0].(*Output).Body.ExtHash = Hash{1}
			},
		},
		{
			desc: "misordered tx time range",
			f: func() {
				tx.Body.MinTimeMS = tx.Body.MaxTimeMS + 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "empty tx results",
			f: func() {
				tx.Body.ResultIDs = nil
			},
			err: errEmptyResults,
		},
		{
			desc: "empty tx results, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Body.ResultIDs = nil
			},
		},
		{
			desc: "tx header exthash nonempty",
			f: func() {
				tx.Body.ExtHash = Hash{1}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "tx header exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Body.ExtHash = Hash{1}
			},
		},
		{
			desc: "wrong blockchain",
			f: func() {
				vs.blockchainID = Hash{2}
			},
			err: errWrongBlockchain,
		},
		{
			desc: "issuance asset ID mismatch",
			f: func() {
				iss := tx.TxInputs[0].(*Issuance)
				iss.Body.Value.AssetID = AssetID{1}
			},
			err: errMismatchedAssetID,
		},
		{
			desc: "issuance program failure",
			f: func() {
				iss := tx.TxInputs[0].(*Issuance)
				iss.Witness.Arguments[0] = []byte{}
			},
			err: vm.ErrFalseVMResult,
		},
		// TODO(bobg): more validation tests
	}

	for i, c := range cases {
		t.Logf("case %d", i)

		fixture := sample(t, nil)
		tx = NewTx(*fixture.tx).TxEntries
		vs = &validationState{
			blockchainID: fixture.initialBlockID,
			tx:           tx,
		}
		mux = tx.Results[0].(*Output).Body.Source.Entry.(*Mux)

		if c.f != nil {
			c.f()
		}
		err := tx.checkValid(vs)
		if rootErr(err) != c.err {
			t.Errorf("case %d (%s): got error %s, want %s; validationState is:\n%s", i, c.desc, err, c.err, spew.Sdump(vs))
		}
	}
}
