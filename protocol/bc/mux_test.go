package bc

import (
	"testing"

	"chain/protocol/vm"

	"github.com/davecgh/go-spew/spew"
)

func TestMuxValid(t *testing.T) {
	var (
		mux *Mux
		vs  *validationState
	)

	cases := []struct {
		f   func()
		err error
	}{
		{},
		{
			f: func() {
				mux.Body.Program.Code = []byte{byte(vm.OP_FALSE)}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			f: func() {
				mux.Body.Sources[0].Value.Amount++
				mux.Body.Sources[0].Entry.(*Issuance).Witness.Destination.Value.Amount++
			},
			err: errUnbalanced,
		},
		{
			f: func() {
				mux.Body.Sources[0].Value.AssetID = AssetID{255}
				mux.Body.Sources[0].Entry.(*Issuance).Witness.Destination.Value.AssetID = AssetID{255}
			},
			err: errUnbalanced,
		},
	}

	for i, c := range cases {
		t.Logf("case %d", i)

		fixture := sample(t, nil)
		tx := NewTx(*fixture.tx)
		mux = tx.TxEntries.Results[0].(*Output).Body.Source.Entry.(*Mux)
		vs = &validationState{
			blockchainID: fixture.initialBlockID,
			tx:           tx.TxEntries,
			entryID:      tx.TxEntries.Results[0].(*Output).Body.Source.Ref,
		}

		if c.f != nil {
			c.f()
		}
		err := mux.CheckValid(vs)
		if rootErr(err) != c.err {
			t.Errorf("case %d: got error %s, want %s; mux is:\n%s\nvalidationState is:\n%s", i, err, c.err, spew.Sdump(mux), spew.Sdump(vs))
		}
	}
}
