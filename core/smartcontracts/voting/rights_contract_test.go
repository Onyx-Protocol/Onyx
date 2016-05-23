package voting

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"chain/core/asset/assettest"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/cos/txscript/txscripttest"
	"chain/crypto/hash256"
	"chain/database/pg/pgtest"
)

const (
	// mockTimeUnix is the unix timestamp to be used as the current
	// time while running scripts in these tests.
	mockTimeUnix = 1
)

var (
	exampleHash  bc.Hash
	exampleHash2 bc.Hash
	mockTimeFunc = func(vm *txscript.Engine) {
		vm.SetTimestamp(time.Unix(mockTimeUnix, 0))
	}
)

func init() {
	var err error
	exampleHash, err = bc.ParseHash("9414886b1ebf025db067a4cbd13a0903fbd9733a5372bba1b58bd72c1699b798")
	if err != nil {
		panic(err)
	}
	exampleHash2, err = bc.ParseHash("cbf9cf4baf8d5636383f5d1412e8ebecc977c1a855f70a63cca4ff7416128532")
	if err != nil {
		panic(err)
	}
}

func TestAuthenticateClause(t *testing.T) {
	ctx := pgtest.NewContext(t)

	var (
		assetID     = assettest.CreateAssetFixture(ctx, t, "", "", "")
		assetAmount = bc.AssetAmount{AssetID: assetID, Amount: 1}
	)

	testCases := []struct {
		err  error
		prev rightScriptData
		out  rightScriptData
	}{
		{
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
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			err: nil,
			prev: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    false,
				OwnershipChain: exampleHash,
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    false,
				OwnershipChain: exampleHash,
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			// Fails because the deadline is in the past.
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
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			// Fails because the deadline changed in the output.
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
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			// Fails because the delegatable field changed in the output.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    false,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			// Fails because the ownership chain changed in the output.
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
				OwnershipChain: exampleHash,
				HolderScript:   []byte{txscript.OP_1},
			},
		},
		{
			// Fails because the holder script changed in the output.
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
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1},
			},
		},
		{
			// Fails during the EVAL of the holder script because the
			// holder script is an unspendable address.
			err: txscript.ErrStackEarlyReturn,
			prev: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_RETURN},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_RETURN},
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder().
			AddInt64(int64(clauseAuthenticate)).
			AddData(rightsHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(assetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(assetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestTransferClause(t *testing.T) {
	ctx := pgtest.NewContext(t)

	var (
		assetID     = assettest.CreateAssetFixture(ctx, t, "", "", "")
		assetAmount = bc.AssetAmount{AssetID: assetID, Amount: 1}
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       190000000,
				Delegatable:    false,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: exampleHash,
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
	}

	for i, tc := range testCases {
		sigBuilder := txscript.NewScriptBuilder()
		sigBuilder = sigBuilder.
			AddData(tc.out.HolderScript).
			AddInt64(int64(clauseTransfer)).
			AddData(rightsHoldingContract)
		sigscript, err := sigBuilder.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(assetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(assetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestDelegateClause(t *testing.T) {
	ctx := pgtest.NewContext(t)

	var (
		assetID     = assettest.CreateAssetFixture(ctx, t, "", "", "")
		assetAmount = bc.AssetAmount{AssetID: assetID, Amount: 1}
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       2,
				Delegatable:    false,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       0,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 0),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       4,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
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
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       3,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(exampleHash, []byte{txscript.OP_1}, 3),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
	}

	for i, tc := range testCases {
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

		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(assetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(assetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestRecallClause(t *testing.T) {
	ctx := pgtest.NewContext(t)

	var (
		assetID     = assettest.CreateAssetFixture(ctx, t, "", "", "")
		assetAmount = bc.AssetAmount{AssetID: assetID, Amount: 1}
	)

	testCases := []struct {
		err          error
		intermediate []bc.Hash
		prev         rightScriptData
		utxo         rightScriptData
		out          rightScriptData
	}{
		{
			// Direct recall
			err:          nil,
			intermediate: []bc.Hash{}, // no intermediate custodians
			prev: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
			utxo: rightScriptData{
				Deadline:       1700000000,
				Delegatable:    false,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 1858259488),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_VERIFY},
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1858259488,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// One intermediate hash in chain of ownership
			err: nil,
			intermediate: []bc.Hash{
				hash256.Sum(append(
					asByteSlice(hash256.Sum(exampleHash[:])),
					asByteSlice(hash256.Sum(txscript.Int64ToScriptBytes(0x7eeeeeee)))...,
				)),
			},
			prev: rightScriptData{
				Deadline:       0x7fffffff,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
			utxo: rightScriptData{
				Deadline:    0x7ddddddd,
				Delegatable: false,
				OwnershipChain: calculateOwnershipChain(
					calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 0x7fffffff),
					exampleHash[:],
					0x7eeeeeee,
				),
				HolderScript: []byte{txscript.OP_RETURN},
				AdminScript:  []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       0x7fffffff,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// Two intermediate hashes in chain of ownership
			err: nil,
			intermediate: []bc.Hash{
				hash256.Sum(append(
					asByteSlice(hash256.Sum([]byte("another holder script"))),
					asByteSlice(hash256.Sum(txscript.Int64ToScriptBytes(0x7eeeeeee)))...,
				)),
				hash256.Sum(append(
					asByteSlice(hash256.Sum(exampleHash[:])),
					asByteSlice(hash256.Sum(txscript.Int64ToScriptBytes(0x7ddddddd)))...,
				)),
			},
			prev: rightScriptData{
				Deadline:       0x7fffffff,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
			utxo: rightScriptData{
				Deadline:    0x7ccccccc,
				Delegatable: false,
				OwnershipChain: calculateOwnershipChain(
					calculateOwnershipChain(
						calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 0x7fffffff),
						exampleHash[:],
						0x7ddddddd,
					),
					[]byte("another holder script"),
					0x7eeeeeee,
				),
				HolderScript: []byte{txscript.OP_RETURN},
				AdminScript:  []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       0x7fffffff,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// Different (earlier) deadline than recall point.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       0x6fffffff,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
			utxo: rightScriptData{
				Deadline:       0x6ccccccc,
				Delegatable:    false,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, 0x6fffffff),
				HolderScript:   []byte{txscript.OP_RETURN},
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       0x7fffffff,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// Holder doesn't authorize.
			err: txscript.ErrStackVerifyFailed,
			prev: rightScriptData{
				Deadline:       0x7fffffff,
				Delegatable:    false,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_RETURN},
				AdminScript:    []byte{txscript.OP_1},
			},
			utxo: rightScriptData{
				Deadline:       0x7ccccccc,
				Delegatable:    false,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_RETURN}, 0x7fffffff),
				HolderScript:   []byte{txscript.OP_RETURN},
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       0x7fffffff,
				Delegatable:    false,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_RETURN},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		for _, h := range tc.intermediate {
			sb.AddData(h[:])
		}
		sb = sb.
			AddInt64(int64(len(tc.intermediate))).
			AddData(tc.prev.HolderScript).
			AddInt64(tc.prev.Deadline).
			AddData(tc.prev.OwnershipChain[:]).
			AddInt64(int64(clauseRecall)).
			AddData(rightsHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(assetAmount, tc.utxo.PKScript(), sigscript).
			AddOutput(assetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestOverrideClause(t *testing.T) {
	ctx := pgtest.NewContext(t)

	var (
		assetID     = assettest.CreateAssetFixture(ctx, t, "", "", "")
		assetAmount = bc.AssetAmount{AssetID: assetID, Amount: 1}
	)

	testCases := []struct {
		err         error
		newHolders  []RightHolder
		proofHashes []bc.Hash
		forkHash    bc.Hash
		utxo        rightScriptData
		out         rightScriptData
	}{
		{
			// 1-level delegate from original holder
			err: nil,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP}, Deadline: infiniteDeadline}, // new holder
				{Script: []byte{txscript.OP_1}, Deadline: infiniteDeadline},                                  // original holder
			},
			proofHashes: []bc.Hash{},
			forkHash:    bc.Hash{}, // 0x00...00
			utxo: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// 2-level delegate from original holder
			err: nil,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP}, Deadline: 100}, // new holder
				{Script: []byte{txscript.OP_RETURN}, Deadline: 1000},                            // intermediate holder
				{Script: []byte{txscript.OP_1}, Deadline: infiniteDeadline},                     // original holder
			},
			proofHashes: []bc.Hash{},
			forkHash:    bc.Hash{}, // 0x00...00
			utxo: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       100,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline), []byte{txscript.OP_RETURN}, 1000),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// recall to original holder
			err: nil,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_1}, Deadline: infiniteDeadline}, // original holder
			},
			proofHashes: []bc.Hash{
				RightHolder{Script: []byte{txscript.OP_1}, Deadline: infiniteDeadline}.hash(),
			},
			forkHash: bc.Hash{}, // 0x00...00
			utxo: rightScriptData{
				Deadline:       1000,
				Delegatable:    false,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
				HolderScript:   []byte{txscript.OP_RETURN},
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// recall to not original holder
			err: nil,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_0, txscript.OP_DROP, txscript.OP_1}, Deadline: 1000}, // recall holder
			},
			proofHashes: []bc.Hash{
				RightHolder{Script: []byte{txscript.OP_0, txscript.OP_DROP, txscript.OP_1}, Deadline: 1000}.hash(), // recall holder
			},
			forkHash: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
			utxo: rightScriptData{
				Deadline:    100,
				Delegatable: false,
				OwnershipChain: calculateOwnershipChain(calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
					[]byte{txscript.OP_0, txscript.OP_DROP, txscript.OP_1}, 1000),
				HolderScript: []byte{txscript.OP_RETURN},
				AdminScript:  []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       1000,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
				HolderScript:   []byte{txscript.OP_0, txscript.OP_DROP, txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
		{
			// transfer
			//
			//              +------------------------> OP_RETURN  (current holder)
			//              |
			// OP_1 -> OP_0 OP_DROP OP_1 (forkhash)
			//              |
			//              +------------------------> OP_0 OP_1ADD (new holder)
			err: nil,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_0, txscript.OP_1ADD}, Deadline: 1000}, // new holder
			},
			proofHashes: []bc.Hash{},
			forkHash: calculateOwnershipChain(calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
				[]byte{txscript.OP_0, txscript.OP_DROP, txscript.OP_1}, 1000),
			utxo: rightScriptData{
				Deadline:    100,
				Delegatable: false,
				OwnershipChain: calculateOwnershipChain(calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
					[]byte{txscript.OP_0, txscript.OP_DROP, txscript.OP_1}, 1000),
				HolderScript: []byte{txscript.OP_RETURN},
				AdminScript:  []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:    1000,
				Delegatable: true,
				OwnershipChain: calculateOwnershipChain(calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
					[]byte{txscript.OP_0, txscript.OP_DROP, txscript.OP_1}, 1000),
				HolderScript: []byte{txscript.OP_0, txscript.OP_1ADD},
				AdminScript:  []byte{txscript.OP_1},
			},
		},
		{
			// multi-level rewrite
			//
			//           +------------> OP_2 -> OP_3 -> OP_4 (current holder)
			//           |
			// OP_0 -> OP_1 (forkhash)
			//           |
			//           +------------> OP_5 -> OP_6 (new holder)
			err: nil,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_6}, Deadline: 10},  // new holder
				{Script: []byte{txscript.OP_5}, Deadline: 100}, // new intermediary holder
			},
			proofHashes: []bc.Hash{
				RightHolder{Script: []byte{txscript.OP_3}, Deadline: 10}.hash(),
				RightHolder{Script: []byte{txscript.OP_2}, Deadline: 100}.hash(),
			},
			forkHash: calculateOwnershipChain(
				calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_0}, infiniteDeadline),
				[]byte{txscript.OP_1}, 1000,
			),
			utxo: rightScriptData{
				Deadline:    1,
				Delegatable: false,
				OwnershipChain: calculateOwnershipChain(
					calculateOwnershipChain(
						calculateOwnershipChain(
							calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_0}, infiniteDeadline),
							[]byte{txscript.OP_1}, 1000,
						),
						[]byte{txscript.OP_2}, 100,
					),
					[]byte{txscript.OP_3}, 10,
				),
				HolderScript: []byte{txscript.OP_4},
				AdminScript:  []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:    10,
				Delegatable: true,
				OwnershipChain: calculateOwnershipChain(
					calculateOwnershipChain(
						calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_0}, infiniteDeadline),
						[]byte{txscript.OP_1}, 1000,
					),
					[]byte{txscript.OP_5}, 100,
				),
				HolderScript: []byte{txscript.OP_6},
				AdminScript:  []byte{txscript.OP_1},
			},
		},
		{
			// admin must authorize override
			err: txscript.ErrStackScriptFailed,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP}, Deadline: infiniteDeadline}, // new holder
				{Script: []byte{txscript.OP_1}, Deadline: infiniteDeadline},                                  // original holder
			},
			proofHashes: []bc.Hash{},
			forkHash:    bc.Hash{}, // 0x00...00
			utxo: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_0, txscript.OP_0},
			},
			out: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_1}, infiniteDeadline),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP},
				AdminScript:    []byte{txscript.OP_0, txscript.OP_0},
			},
		},
		{
			// can't change original holder
			err: txscript.ErrStackVerifyFailed,
			newHolders: []RightHolder{
				{Script: []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP}, Deadline: infiniteDeadline}, // new holder
				{Script: []byte{txscript.OP_RETURN}, Deadline: infiniteDeadline},                             // original holder
			},
			proofHashes: []bc.Hash{},
			forkHash:    bc.Hash{}, // 0x00...00
			utxo: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: bc.Hash{}, // 0x000...000
				HolderScript:   []byte{txscript.OP_1},
				AdminScript:    []byte{txscript.OP_1},
			},
			out: rightScriptData{
				Deadline:       infiniteDeadline,
				Delegatable:    true,
				OwnershipChain: calculateOwnershipChain(bc.Hash{}, []byte{txscript.OP_RETURN}, infiniteDeadline),
				HolderScript:   []byte{txscript.OP_1, txscript.OP_1, txscript.OP_DROP},
				AdminScript:    []byte{txscript.OP_1},
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		for _, h := range tc.newHolders {
			sb.AddData(h.Script)
			sb.AddInt64(h.Deadline)
		}
		sb.AddInt64(int64(len(tc.newHolders)))
		for _, h := range tc.proofHashes {
			sb.AddData(h[:])
		}
		sb.AddInt64(int64(len(tc.proofHashes))).
			AddData(tc.forkHash[:]).
			AddBool(tc.out.Delegatable).
			AddInt64(int64(clauseOverride)).
			AddData(rightsHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(assetAmount, tc.utxo.PKScript(), sigscript).
			AddOutput(assetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

// TestRightsContractValidMatch tests generating a pkscript from a voting right.
// The generated pkscript is then used in the voting rights p2c detection
// flow, where it should be found to match the contract. Then the decoded
// voting right and the original voting right are checked for equality.
func TestRightsContractValidMatch(t *testing.T) {
	testCases := []rightScriptData{
		{
			AdminScript:    []byte{txscript.OP_1},
			HolderScript:   []byte{0xde, 0xad, 0xbe, 0xef},
			OwnershipChain: exampleHash,
			Deadline:       1457988220,
			Delegatable:    true,
		},
		{
			AdminScript:    []byte{txscript.OP_1},
			HolderScript:   []byte{},
			OwnershipChain: exampleHash,
			Deadline:       1457988221,
			Delegatable:    false,
		},
		{
			AdminScript:    []byte{txscript.OP_1},
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

		if !bytes.Equal(got.AdminScript, want.AdminScript) {
			t.Errorf("%d: Right.AdminScript, got=%#v want=%#v", i, got.AdminScript, want.AdminScript)
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
			[]byte{}, []byte{}, []byte{}, []byte{}, []byte{},
		},
		{ // chain of ownership hash not long enough
			[]byte{0x01},                   // delegatable = true
			[]byte{0x56, 0xE7, 0x2C, 0xC8}, // deadline = 1457990856
			[]byte{0xde, 0xad, 0xbe, 0xef}, // ownership chain hash = 0xdeadbeef
			[]byte{0xde, 0xad, 0xbe, 0xef}, // holding script = 0xdeadbeef
			[]byte{0xde, 0xad, 0xbe, 0xef}, // admin script = 0xdeadbeef
		},
		{ // five parameter input
			[]byte{0x00},                   // delegatable = false
			[]byte{0x56, 0xE7, 0x2C, 0xC8}, // deadline = 1457990856
			exampleHash[:],                 // ownership chain hash = 0x9414..98
			[]byte{0xde, 0xad, 0xbe, 0xef}, // holding script = 0xdeadbeef
			[]byte{0xde, 0xad, 0xbe, 0xef}, // admin script = 0xdeadbeef
			[]byte{0x02},                   // extra parameter on the end
		},
	}

	for i, params := range testCaseScriptParams {
		script, err := txscript.PayToContractHash(rightsHoldingContractHash, params, scriptVersion)
		if err != nil {
			t.Errorf("%d: building pkscript: %s", i, err)
			continue
		}

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

func asByteSlice(h [hash256.Size]byte) []byte {
	return h[:]
}
