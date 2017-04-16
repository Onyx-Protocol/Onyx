package validation

import (
	"math"
	"testing"
	"time"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/vm"
	"chain/testutil"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
)

func init() {
	spew.Config.DisableMethods = true
}

func TestTxValidation(t *testing.T) {
	var (
		tx      *bc.Tx
		vs      *validationState
		fixture *txFixture

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
				iss := tx.Entries[*mux.Body.Sources[0].Ref].(*bc.Issuance)
				iss.Witness.Destination.Value.Amount++
			},
			err: errUnbalanced,
		},
		{
			desc: "overflowing mux source amounts",
			f: func() {
				mux.Body.Sources[0].Value.Amount = math.MaxInt64
				iss := tx.Entries[*mux.Body.Sources[0].Ref].(*bc.Issuance)
				iss.Witness.Destination.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "underflowing mux destination amounts",
			f: func() {
				mux.Witness.Destinations[0].Value.Amount = math.MaxInt64
				out := tx.Entries[*mux.Witness.Destinations[0].Ref].(*bc.Output)
				out.Body.Source.Value.Amount = math.MaxInt64
				mux.Witness.Destinations[1].Value.Amount = math.MaxInt64
				out = tx.Entries[*mux.Witness.Destinations[1].Ref].(*bc.Output)
				out.Body.Source.Value.Amount = math.MaxInt64
			},
			err: errOverflow,
		},
		{
			desc: "unbalanced mux assets",
			f: func() {
				mux.Body.Sources[1].Value.AssetId = newAssetID(255)
				sp := tx.Entries[*mux.Body.Sources[1].Ref].(*bc.Spend)
				sp.Witness.Destination.Value.AssetId = newAssetID(255)
			},
			err: errUnbalanced,
		},
		{
			desc: "nonempty mux exthash",
			f: func() {
				mux.Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonempty mux exthash, but that's OK",
			f: func() {
				tx.Body.Version = 2
				mux.Body.ExtHash = newHash(1)
			},
		},
		{
			desc: "failing nonce program",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := tx.Entries[*iss.Body.AnchorId].(*bc.Nonce)
				nonce.Body.Program.Code = []byte{byte(vm.OP_FALSE)}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "nonce exthash nonempty",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := tx.Entries[*iss.Body.AnchorId].(*bc.Nonce)
				nonce.Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonce exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := tx.Entries[*iss.Body.AnchorId].(*bc.Nonce)
				nonce.Body.ExtHash = newHash(1)
			},
		},
		{
			desc: "nonce timerange misordered",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := tx.Entries[*iss.Body.AnchorId].(*bc.Nonce)
				tr := tx.Entries[*nonce.Body.TimeRangeId].(*bc.TimeRange)
				tr.Body.MinTimeMs = tr.Body.MaxTimeMs + 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "nonce timerange disagrees with tx timerange",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := tx.Entries[*iss.Body.AnchorId].(*bc.Nonce)
				tr := tx.Entries[*nonce.Body.TimeRangeId].(*bc.TimeRange)
				tr.Body.MaxTimeMs = tx.Body.MaxTimeMs - 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "nonce timerange exthash nonempty",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := tx.Entries[*iss.Body.AnchorId].(*bc.Nonce)
				tr := tx.Entries[*nonce.Body.TimeRangeId].(*bc.TimeRange)
				tr.Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "nonce timerange exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				iss := tx.TxInputs[0].(*bc.Issuance)
				nonce := tx.Entries[*iss.Body.AnchorId].(*bc.Nonce)
				tr := tx.Entries[*nonce.Body.TimeRangeId].(*bc.TimeRange)
				tr.Body.ExtHash = newHash(1)
			},
		},
		{
			desc: "mismatched output source / mux dest position",
			f: func() {
				tx.Entries[*tx.Body.ResultIds[0]].(*bc.Output).Body.Source.Position = 1
			},
			err: errMismatchedPosition,
		},
		{
			desc: "mismatched output source and mux dest",
			f: func() {
				// For this test, it's necessary to construct a mostly
				// identical second transaction in order to get a similar but
				// not equal output entry for the mux to falsely point
				// to. That entry must be added to the first tx's Entries map.
				fixture.txOutputs[0].ReferenceData = []byte{1}
				fixture2 := sample(t, fixture)
				tx2 := legacy.NewTx(*fixture2.tx).Tx
				out2ID := tx2.Body.ResultIds[0]
				out2 := tx2.Entries[*out2ID].(*bc.Output)
				tx.Entries[*out2ID] = out2
				mux.Witness.Destinations[0].Ref = out2ID
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
				outID := tx.Body.ResultIds[0]
				out := tx.Entries[*outID].(*bc.Output)
				mux.Witness.Destinations[0].Value = &bc.AssetAmount{
					AssetId: out.Body.Source.Value.AssetId,
					Amount:  out.Body.Source.Value.Amount + 1,
				}
				mux.Body.Sources[0].Value.Amount++ // the mux must still balance
			},
			err: errMismatchedValue,
		},
		{
			desc: "output exthash nonempty",
			f: func() {
				tx.Entries[*tx.Body.ResultIds[0]].(*bc.Output).Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "output exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Entries[*tx.Body.ResultIds[0]].(*bc.Output).Body.ExtHash = newHash(1)
			},
		},
		{
			desc: "misordered tx time range",
			f: func() {
				tx.Body.MinTimeMs = tx.Body.MaxTimeMs + 1
			},
			err: errBadTimeRange,
		},
		{
			desc: "empty tx results",
			f: func() {
				tx.Body.ResultIds = nil
			},
			err: errEmptyResults,
		},
		{
			desc: "empty tx results, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Body.ResultIds = nil
			},
		},
		{
			desc: "tx header exthash nonempty",
			f: func() {
				tx.Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "tx header exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.Body.ExtHash = newHash(1)
			},
		},
		{
			desc: "wrong blockchain",
			f: func() {
				vs.blockchainID = *newHash(2)
			},
			err: errWrongBlockchain,
		},
		{
			desc: "issuance asset ID mismatch",
			f: func() {
				iss := tx.TxInputs[0].(*bc.Issuance)
				iss.Body.Value.AssetId = newAssetID(1)
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
				tx.TxInputs[0].(*bc.Issuance).Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "issuance exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.TxInputs[0].(*bc.Issuance).Body.ExtHash = newHash(1)
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
				spentOutput := tx.Entries[*spend.Body.SpentOutputId].(*bc.Output)
				spentOutput.Body.Source.Value = &bc.AssetAmount{
					AssetId: spend.Witness.Destination.Value.AssetId,
					Amount:  spend.Witness.Destination.Value.Amount + 1,
				}
			},
			err: errMismatchedValue,
		},
		{
			desc: "spend exthash nonempty",
			f: func() {
				tx.TxInputs[1].(*bc.Spend).Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
		{
			desc: "spend exthash nonempty, but that's OK",
			f: func() {
				tx.Body.Version = 2
				tx.TxInputs[1].(*bc.Spend).Body.ExtHash = newHash(1)
			},
		},
	}

	for i, c := range cases {
		t.Logf("case %d", i)

		fixture = sample(t, nil)
		tx = legacy.NewTx(*fixture.tx).Tx
		vs = &validationState{
			blockchainID: fixture.initialBlockID,
			tx:           tx,
			entryID:      tx.ID,
		}
		out := tx.Entries[*tx.Body.ResultIds[0]].(*bc.Output)
		muxID := out.Body.Source.Ref
		mux = tx.Entries[*muxID].(*bc.Mux)

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
	base := bc.NewBlockHeader(1, 1, &bc.Hash{}, 1, &bc.Hash{}, &bc.Hash{}, nil)
	baseBytes, _ := proto.Marshal(base)

	var bh bc.BlockHeader

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
				bh.Body.ExtHash = newHash(1)
			},
			err: errNonemptyExtHash,
		},
	}

	for i, c := range cases {
		t.Logf("case %d", i)
		proto.Unmarshal(baseBytes, &bh)
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
	txInputs             []*legacy.TxInput
	txOutputs            []*legacy.TxOutput
	txMinTime, txMaxTime uint64
	txRefData            []byte
	tx                   *legacy.TxData
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

	if result.initialBlockID.IsZero() {
		result.initialBlockID = *newHash(1)
	}
	if testutil.DeepEqual(result.issuanceProg, bc.Program{}) {
		prog, err := vm.Assemble("ADD 5 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		result.issuanceProg = bc.Program{VmVersion: 1, Code: prog}
	}
	if len(result.issuanceArgs) == 0 {
		result.issuanceArgs = [][]byte{[]byte{2}, []byte{3}}
	}
	if len(result.assetDef) == 0 {
		result.assetDef = []byte{2}
	}
	if result.assetID.IsZero() {
		refdatahash := hashData(result.assetDef)
		result.assetID = bc.ComputeAssetID(result.issuanceProg.Code, &result.initialBlockID, result.issuanceProg.VmVersion, &refdatahash)
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

		result.txInputs = []*legacy.TxInput{
			legacy.NewIssuanceInput([]byte{3}, 10, []byte{4}, result.initialBlockID, result.issuanceProg.Code, result.issuanceArgs, result.assetDef),
			legacy.NewSpendInput(args1, *newHash(5), result.assetID, 20, 0, cp1, *newHash(6), []byte{7}),
			legacy.NewSpendInput(args2, *newHash(8), result.assetID, 40, 0, cp2, *newHash(9), []byte{10}),
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

		result.txOutputs = []*legacy.TxOutput{
			legacy.NewTxOutput(result.assetID, 25, cp1, []byte{11}),
			legacy.NewTxOutput(result.assetID, 45, cp2, []byte{12}),
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

	result.tx = &legacy.TxData{
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

func hashData(data []byte) bc.Hash {
	var b32 [32]byte
	sha3pool.Sum256(b32[:], data)
	return bc.NewHash(b32)
}

func newHash(n byte) *bc.Hash {
	h := bc.NewHash([32]byte{n})
	return &h
}

func newAssetID(n byte) *bc.AssetID {
	a := bc.NewAssetID([32]byte{n})
	return &a
}
