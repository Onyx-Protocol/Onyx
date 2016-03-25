package voting

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset/assettest"
	"chain/database/pg/pgtest"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
)

const (
	// mockTimeUnix is the unix timestamp to be used as the current
	// time while running scripts in these tests.
	mockTimeUnix = 1
)

var (
	exampleHash bc.Hash
)

func init() {
	var err error
	exampleHash, err = bc.ParseHash("9414886b1ebf025db067a4cbd13a0903fbd9733a5372bba1b58bd72c1699b798")
	if err != nil {
		panic(err)
	}
}

type utxoTx struct {
	assetID    bc.AssetID
	hash       bc.Hash
	scriptData rightScriptData
	p2cScript  []byte
}

// Output implements the fedchain/txscript.viewReader interface, returning
// the output represented by utxoTx.
func (o utxoTx) Output(ctx context.Context, outpoint bc.Outpoint) *state.Output {
	if outpoint.Hash != o.hash || outpoint.Index != 0 {
		return nil
	}

	return &state.Output{
		TxOutput: bc.TxOutput{
			AssetAmount: bc.AssetAmount{AssetID: o.assetID, Amount: 1},
			Script:      o.p2cScript,
			Metadata:    []byte("utxoTxal voting right outpoint"),
		},
		Outpoint: outpoint,
		Spent:    false,
	}
}

func TestTransferClause(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	var (
		projectID    = assettest.CreateProjectFixture(ctx, t, "", "")
		issuerNodeID = assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
		assetID      = assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	)

	testCases := []struct {
		err  error
		prev rightScriptData
		out  rightScriptData
	}{
		{
			err: nil,
			prev: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
			},
		},
		{
			// Transferring to yourself is OK but pointless.
			err: nil,
			prev: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			// The parameters of the contract can't change during transfer.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       190000000,
				Delegatable:    false,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			// The ownership chain shouldn't change during transfer.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: exampleHash,
				HolderScript:   []byte{txscript.OP_1},
			},
		},
	}

	for i, tc := range testCases {
		utxoTx := utxoTx{
			assetID:    assetID,
			hash:       exampleHash,
			scriptData: tc.prev,
			p2cScript:  tc.prev.PKScript(),
		}

		sigBuilder := txscript.NewScriptBuilder()
		sigBuilder = sigBuilder.
			AddData(tc.out.HolderScript).
			AddInt64(int64(clauseTransfer)).
			AddData(rightsHoldingContract)
		sigscript, err := sigBuilder.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = executeScript(ctx, utxoTx, sigscript, tc.out.PKScript())
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestDelegateClause(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	var (
		projectID    = assettest.CreateProjectFixture(ctx, t, "", "")
		issuerNodeID = assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
		assetID      = assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	)

	testCases := []struct {
		err  error
		prev rightScriptData
		out  rightScriptData
	}{
		{
			// Simple delegate with exact same deadline and delegatable params
			err: nil,
			prev: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
			},
		},
		{
			// Delegate with shorter deadline
			err: nil,
			prev: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       2,
				Delegatable:    false,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
			},
		},
		{
			// Delegate but the deadline already passed
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       0,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       0,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 0),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
			},
		},
		{
			// Shouldn't be able to delegate if the utxo script has
			// Delegatable = false in its contract params.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       3,
				Delegatable:    false,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
			},
		},
		{
			// Delegating with a longer deadline should fail.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       4,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
			},
		},
		{
			// Delegating with a bad ownership chain should fail.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(exampleHash, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
			},
		},
	}

	for i, tc := range testCases {
		utxoTx := utxoTx{
			assetID:    assetID,
			hash:       exampleHash,
			scriptData: tc.prev,
			p2cScript:  tc.prev.PKScript(),
		}

		var delegatable int64 = 0
		if tc.out.Delegatable {
			delegatable = 1
		}

		sb := txscript.NewScriptBuilder().
			AddInt64(tc.out.Deadline).
			AddInt64(delegatable).
			AddData(tc.out.HolderScript).
			AddInt64(int64(clauseDelegate)).
			AddData(rightsHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = executeScript(ctx, utxoTx, sigscript, tc.out.PKScript())
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func executeScript(ctx context.Context, utxoTx utxoTx, sigscript, pkscript []byte) error {
	newTx := bc.TxData{
		Version: 0,
		Inputs: []*bc.TxInput{
			{
				Previous:        bc.Outpoint{Hash: exampleHash, Index: 0},
				SignatureScript: sigscript,
			},
		},
		Outputs: []*bc.TxOutput{
			{
				AssetAmount: bc.AssetAmount{
					AssetID: utxoTx.assetID,
					Amount:  1,
				},
				Script: pkscript,
			},
		},
	}
	vm, err := txscript.NewEngine(ctx, utxoTx, utxoTx.p2cScript, &newTx, 0, 0)
	if err != nil {
		return err
	}
	vm.SetTimestamp(time.Unix(mockTimeUnix, 0))
	return vm.Execute()
}

// TestRightsContractValidMatch tests generating a pkscript from a voting right.
// The generated pkscript is then used in the voting rights p2c detection
// flow, where it should be found to match the contract. Then the decoded
// voting right and the original voting right are checked for equality.
func TestRightsContractValidMatch(t *testing.T) {
	testCases := []rightScriptData{
		{
			HolderScript:   []byte{0xde, 0xad, 0xbe, 0xef},
			OwnershipChain: exampleHash,
			Deadline:       1457988220,
			Delegatable:    true,
		},
		{
			HolderScript:   []byte{},
			OwnershipChain: exampleHash,
			Deadline:       1457988221,
			Delegatable:    false,
		},
		{
			HolderScript:   exampleHash[:],
			OwnershipChain: bc.Hash{}, // 0x00 ... 0x00
			Deadline:       time.Unix(1457988221, 0).AddDate(5, 0, 0).Unix(),
			Delegatable:    true,
		},
	}

	for i, want := range testCases {
		script := want.PKScript()
		got, err := testRightsContract(script)
		if err != nil {
			t.Errorf("%d: testing rights contract for %x: %s", i, script, err)
			continue
		}

		if got == nil {
			t.Errorf("%d: No match for pkscript %x generated from %#v", i, script, want)
			continue
		}

		if !bytes.Equal(got.HolderScript, want.HolderScript) {
			t.Errorf("%d: Right.HolderScript, got=%#v want=%#v", i, got.HolderScript, want.HolderScript)
		}
		if got.OwnershipChain != want.OwnershipChain {
			t.Errorf("%d: Right.OwnershipChain, got=%#v want=%#v", i, got.OwnershipChain, want.OwnershipChain)
		}
		if got.Deadline != want.Deadline {
			t.Errorf("%d: Right.Deadline, got=%#v want=%#v", i, got.Deadline, want.Deadline)
		}
		if got.Delegatable != want.Delegatable {
			t.Errorf("%d: Right.Delegatable, got=%#v want=%#v", i, got.Delegatable, want.Delegatable)
		}
	}
}

// TestRightsContractInvalidScript tests that testRightsContract correctly
// fails on pkscripts that are paid to the rights contract but are
// improperly formatted.
func TestRightsContractInvalidMatch(t *testing.T) {
	testCaseScriptParams := [][][]byte{
		{ // no parameters
		},
		{ // not enough parameters
			[]byte{}, []byte{}, []byte{},
		},
		{ // enough parameters, but all empty
			[]byte{}, []byte{}, []byte{}, []byte{},
		},
		{ // chain of ownership hash not long enough
			[]byte{0x01},                   // delegatable = true
			[]byte{0x56, 0xE7, 0x2C, 0xC8}, // deadline = 1457990856
			[]byte{0xde, 0xad, 0xbe, 0xef}, // ownership chain hash = 0xdeadbeef
			[]byte{0xde, 0xad, 0xbe, 0xef}, // holding script = 0xdeadbeef
		},
		{ // five parameter input
			[]byte{0x00},                   // delegatable = false
			[]byte{0x56, 0xE7, 0x2C, 0xC8}, // deadline = 1457990856
			exampleHash[:],                 // ownership chain hash = 0x9414..98
			[]byte{0xde, 0xad, 0xbe, 0xef}, // holding script = 0xdeadbeef
			[]byte{0x02},                   // extra parameter on the end
		},
	}

	for i, params := range testCaseScriptParams {
		addr := txscript.NewAddressContractHash(rightsHoldingContractHash[:], scriptVersion, params)
		script := addr.ScriptAddress()

		data, err := testRightsContract(script)
		if err != nil {
			t.Errorf("%d: testing rights contract for %x: %s", i, script, err)
			continue
		}

		if data != nil {
			t.Errorf("%d: Match for pkscript %x generated from params: %#v", i, script, params)
			continue
		}
	}
}
