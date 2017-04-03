package validation

import (
	"math"
	"testing"
	"time"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/testutil"

	"github.com/davecgh/go-spew/spew"
)

func TestTxValidation(t *testing.T) {
	var (
		tx *bc.TxEntries
		vs *validationState

		// the mux from tx, pulled out for convenience
		mux *bc.Mux
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
				mux.Body.Sources[0].Entry.(*bc.Issuance).Witness.Destination.Value.Amount++
			},
			err: errUnbalanced,
		},
		{
			desc: "overflowing mux source amounts",
			f: func() {
				mux.Body.Sources[0].Value.Amount = math.MaxInt64
				mux.Body.Sources[0].Entry.(*bc.Issuance).Witness.Destination.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "underflowing mux destination amounts",
			f: func() {
				mux.Witness.Destinations[0].Value.Amount = math.MaxInt64
				mux.Witness.Destinations[0].Entry.(*bc.Output).Body.Source.Value.Amount = math.MaxInt64
				mux.Witness.Destinations[1].Value.Amount = math.MaxInt64
				mux.Witness.Destinations[1].Entry.(*bc.Output).Body.Source.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "unbalanced mux assets",
			f: func() {
				mux.Body.Sources[0].Value.AssetID = bc.AssetID{255}
				mux.Body.Sources[0].Entry.(*bc.Issuance).Witness.Destination.Value.AssetID = bc.AssetID{255}
			},
			err: errUnbalanced,
		},
		{
			desc: "nonempty mux exthash",
			f: func() {
				mux.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonempty mux exthash, but that's OK",
			f: func() {
				tx.Body.Version = 2
				mux.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
		},
		{
			desc: "failing nonce program",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := iss.Anchor.(*bc.Nonce)
				nonce.Body.Program.Code = []byte{byte(vm.OP_FALSE)}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "nonce exthash nonempty",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := iss.Anchor.(*bc.Nonce)
				nonce.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonce exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := iss.Anchor.(*bc.Nonce)
				nonce.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
		},
		{
			desc: "nonce timerange misordered",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := iss.Anchor.(*bc.Nonce)
				tr := nonce.TimeRange
				tr.Body.MinTimeMS = tr.Body.MaxTimeMS + 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "nonce timerange disagrees with tx timerange",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := iss.Anchor.(*bc.Nonce)
				tr := nonce.TimeRange
				tr.Body.MaxTimeMS = tx.Body.MaxTimeMS - 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "nonce timerange exthash nonempty",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := iss.Anchor.(*bc.Nonce)
				tr := nonce.TimeRange
				tr.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonce timerange exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := iss.Anchor.(*bc.Nonce)
				tr := nonce.TimeRange
				tr.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
		},
		{
			desc: "mismatched output source / mux dest position",
			f: func() {
				tx.Results[0].(*bc.Output).Body.Source.Position = 1
			},
			err: errMismatchedPosition,
		},
		{
			desc: "mismatched output source and mux dest",
			f: func() {
				mux.Witness.Destinations[0].Ref = bc.Hash{0x0100000000000000, 0, 0, 0}
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
				mux.Witness.Destinations[0].Value.Amount = tx.Results[0].(*bc.Output).Body.Source.Value.Amount + 1
			},
			err: errMismatchedValue,
		},
		{
			desc: "output exthash nonempty",
			f: func() {
				tx.Results[0].(*bc.Output).Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "output exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Results[0].(*bc.Output).Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
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
				tx.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "tx header exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
		},
		{
			desc: "wrong blockchain",
			f: func() {
				vs.blockchainID = bc.Hash{0x0200000000000000, 0, 0, 0}
			},
			err: errWrongBlockchain,
		},
		{
			desc: "issuance asset ID mismatch",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				iss.Body.Value.AssetID = bc.AssetID{1}
			},
			err: errMismatchedAssetID,
		},
		{
			desc: "issuance program failure",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				iss.Witness.Arguments[0] = []byte{}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "issuance exthash nonempty",
			f: func() {
				tx.TxInputs[0].(*bc.Issuance).Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "issuance exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.TxInputs[0].(*bc.Issuance).Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
		},
		{
			desc: "spend control program failure",
			f: func() {
				tx.TxInputs[1].(*bc.Spend).Witness.Arguments[0] = []byte{}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "mismatched spent source/witness value",
			f: func() {
				spend := tx.TxInputs[1].(*bc.Spend)
				spend.SpentOutput.Body.Source.Value.Amount = spend.Witness.Destination.Value.Amount + 1
			},
			err: errMismatchedValue,
		},
		{
			desc: "spend exthash nonempty",
			f: func() {
				tx.TxInputs[1].(*bc.Spend).Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "spend exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.TxInputs[1].(*bc.Spend).Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
		},
	}

	for i, c := range cases {
		t.Logf("case %d", i)

		fixture := sample(t, nil)
		tx = bc.NewTx(*fixture.tx).TxEntries
		vs = &validationState{
			blockchainID: fixture.initialBlockID,
			tx:           tx,
			entryID:      tx.ID,
		}
		mux = tx.Results[0].(*bc.Output).Body.Source.Entry.(*bc.Mux)

		if c.f != nil {
			c.f()
		}
		err := checkValid(vs, tx.TxHeader)
		if rootErr(err) != c.err {
			t.Errorf("case %d (%s): got error %s, want %s; validationState is:\n%s", i, c.desc, err, c.err, spew.Sdump(vs))
		}
	}
}

func TestBlockHeaderValid(t *testing.T) {
	base := bc.NewBlockHeaderEntry(1, 1, bc.Hash{}, 1, bc.Hash{}, bc.Hash{}, nil)

	var bh bc.BlockHeaderEntry

	cases := []struct {
		f   func()
		err error
	}{
		{},
		{
			f: func() {
				bh.Body.Version = 2
			},
		},
		{
			f: func() {
				bh.Body.ExtHash = bc.Hash{0x0100000000000000, 0, 0, 0}
			},
			err: errNonemptyExtHash,
		},
	}

	for i, c := range cases {
		t.Logf("case %d", i)
		bh = *base
		if c.f != nil {
			c.f()
		}
		err := checkValidBlockHeader(&bh)
		if err != c.err {
			t.Errorf("case %d: got error %s, want %s; bh is:\n%s", i, err, c.err, spew.Sdump(bh))
		}
	}
}

// A txFixture is returned by sample (below) to produce a sample
// transaction, which takes a separate, optional _input_ txFixture to
// affect the transaction that's built. The components of the
// transaction are the fields of txFixture.
type txFixture struct {
	initialBlockID       bc.Hash
	issuanceProg         bc.Program
	issuanceArgs         [][]byte
	assetDef             []byte
	assetID              bc.AssetID
	txVersion            uint64
	txInputs             []*bc.TxInput
	txOutputs            []*bc.TxOutput
	txMinTime, txMaxTime uint64
	txRefData            []byte
	tx                   *bc.TxData
}

// Produces a sample transaction in a txFixture object (see above). A
// separate input txFixture can be used to alter the transaction
// that's created.
//
// The output of this function can be used as the input to a
// subsequent call to make iterative refinements to a test object.
//
// The default transaction produced is valid and has three inputs:
//  - an issuance of 10 units
//  - a spend of 20 units
//  - a spend of 40 units
// and two outputs, one of 25 units and one of 45 units.
// All amounts are denominated in the same asset.
//
// The issuance program for the asset requires two numbers as
// arguments that add up to 5. The prevout control programs require
// two numbers each, adding to 9 and 13, respectively.
//
// The min and max times for the transaction are now +/- one minute.
func sample(tb testing.TB, in *txFixture) *txFixture {
	var result txFixture
	if in != nil {
		result = *in
	}

	if (result.initialBlockID == bc.Hash{}) {
		result.initialBlockID = bc.Hash{0x0100000000000000, 0, 0, 0}
	}
	if testutil.DeepEqual(result.issuanceProg, bc.Program{}) {
		prog, err := vm.Assemble("ADD 5 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		result.issuanceProg = bc.Program{VMVersion: 1, Code: prog}
	}
	if len(result.issuanceArgs) == 0 {
		result.issuanceArgs = [][]byte{[]byte{2}, []byte{3}}
	}
	if len(result.assetDef) == 0 {
		result.assetDef = []byte{2}
	}
	if (result.assetID == bc.AssetID{}) {
		result.assetID = bc.ComputeAssetID(result.issuanceProg.Code, result.initialBlockID, result.issuanceProg.VMVersion, hashData(result.assetDef))
	}

	if result.txVersion == 0 {
		result.txVersion = 1
	}
	if len(result.txInputs) == 0 {
		cp1, err := vm.Assemble("ADD 9 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args1 := [][]byte{[]byte{4}, []byte{5}}

		cp2, err := vm.Assemble("ADD 13 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args2 := [][]byte{[]byte{6}, []byte{7}}

		result.txInputs = []*bc.TxInput{
			bc.NewIssuanceInput([]byte{3}, 10, []byte{4}, result.initialBlockID, result.issuanceProg.Code, result.issuanceArgs, result.assetDef),
			bc.NewSpendInput(args1, bc.Hash{0x0500000000000000, 0, 0, 0}, result.assetID, 20, 0, cp1, bc.Hash{0x0600000000000000, 0, 0, 0}, []byte{7}),
			bc.NewSpendInput(args2, bc.Hash{0x0800000000000000, 0, 0, 0}, result.assetID, 40, 0, cp2, bc.Hash{0x900000000000000, 0, 0, 0}, []byte{10}),
		}
	}
	if len(result.txOutputs) == 0 {
		cp1, err := vm.Assemble("ADD 17 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		cp2, err := vm.Assemble("ADD 21 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}

		result.txOutputs = []*bc.TxOutput{
			bc.NewTxOutput(result.assetID, 25, cp1, []byte{11}),
			bc.NewTxOutput(result.assetID, 45, cp2, []byte{12}),
		}
	}
	if result.txMinTime == 0 {
		result.txMinTime = bc.Millis(time.Now().Add(-time.Minute))
	}
	if result.txMaxTime == 0 {
		result.txMaxTime = bc.Millis(time.Now().Add(time.Minute))
	}
	if len(result.txRefData) == 0 {
		result.txRefData = []byte{13}
	}

	result.tx = &bc.TxData{
		Version:       result.txVersion,
		Inputs:        result.txInputs,
		Outputs:       result.txOutputs,
		MinTime:       result.txMinTime,
		MaxTime:       result.txMaxTime,
		ReferenceData: result.txRefData,
	}

	return &result
}

// Like errors.Root, but also unwraps vm.Error objects.
func rootErr(e error) error {
	for {
		e = errors.Root(e)
		if e2, ok := e.(vm.Error); ok {
			e = e2.Err
			continue
		}
		return e
	}
}

func hashData(data []byte) (h bc.Hash) {
	var b32 bc.Byte32
	sha3pool.Sum256(b32[:], data)
	h.FromByte32(b32)
	return h
}
