package voting

import (
	"bytes"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/core/asset/assettest"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/cos/txscript/txscripttest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestRedistributeClause(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		rightA  = assettest.CreateAssetFixture(ctx, t, "", "", "")
		rightB  = assettest.CreateAssetFixture(ctx, t, "", "", "")
		rightC  = assettest.CreateAssetFixture(ctx, t, "", "", "")
		rightD  = assettest.CreateAssetFixture(ctx, t, "", "", "")
		rightE  = assettest.CreateAssetFixture(ctx, t, "", "", "")
		assetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		right   = rightScriptData{
			Delegatable:    true,
			OwnershipChain: bc.Hash{}, // 0x000...000
			HolderScript:   []byte{txscript.OP_1},
		}
	)

	testCases := []struct {
		err           error
		amount        uint64
		distributions map[bc.AssetID]uint64
		prev          tokenScriptData
		outs          map[*tokenScriptData]uint64
	}{
		{
			// Redistribute to four different voting rights.
			amount: 10000,
			distributions: map[bc.AssetID]uint64{
				rightB: 1000,
				rightC: 500,
				rightD: 500,
				rightE: 4000,
			},
			prev: tokenScriptData{
				Right: rightA,
				State: stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{Right: rightB}: 1000,
				&tokenScriptData{Right: rightC}: 500,
				&tokenScriptData{Right: rightD}: 500,
				&tokenScriptData{Right: rightE}: 4000,
				&tokenScriptData{Right: rightA}: 4000,
			},
		},
		{
			// Redistribute nothing, everything is change.
			amount:        10000,
			distributions: map[bc.AssetID]uint64{},
			prev: tokenScriptData{
				Right: rightA,
				State: stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{Right: rightA}: 10000,
			},
		},
		{
			// Token must be in distributed state.
			err:    txscript.ErrStackVerifyFailed,
			amount: 10000,
			distributions: map[bc.AssetID]uint64{
				rightB: 1000,
				rightC: 500,
				rightD: 500,
				rightE: 4000,
			},
			prev: tokenScriptData{
				Right: rightA,
				State: stateRegistered,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{Right: rightB}: 1000,
				&tokenScriptData{Right: rightC}: 500,
				&tokenScriptData{Right: rightD}: 500,
				&tokenScriptData{Right: rightE}: 4000,
				&tokenScriptData{Right: rightA}: 4000,
			},
		},
		{
			// Distribution amounts don't match outputs.
			err:    txscript.ErrStackVerifyFailed,
			amount: 10000,
			distributions: map[bc.AssetID]uint64{
				rightB: 1000,
				rightC: 500,
				rightD: 500,
				rightE: 4000,
			},
			prev: tokenScriptData{
				Right: rightA,
				State: stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{Right: rightB}: 1001,
				&tokenScriptData{Right: rightC}: 499,
				&tokenScriptData{Right: rightD}: 500,
				&tokenScriptData{Right: rightE}: 4000,
				&tokenScriptData{Right: rightA}: 4000,
			},
		},
		{
			// Change goes to a voting right different from the input.
			err:    txscript.ErrStackScriptFailed,
			amount: 10000,
			distributions: map[bc.AssetID]uint64{
				rightB: 1000,
				rightC: 500,
				rightD: 500,
				rightE: 4000,
			},
			prev: tokenScriptData{
				Right: rightA,
				State: stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{Right: rightB}: 1000,
				&tokenScriptData{Right: rightC}: 500,
				&tokenScriptData{Right: rightD}: 500,
				&tokenScriptData{Right: rightE}: 4000,
				&tokenScriptData{Right: rightB}: 4000,
			},
		},
	}
	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		for r, amt := range tc.distributions {
			sb = sb.AddInt64(int64(amt)).AddData(r[:])
		}
		sb = sb.
			AddInt64(int64(len(tc.distributions))).
			AddData(right.PKScript()).
			AddInt64(int64(clauseRedistribute)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		tx := txscripttest.NewTestTx(mockTimeFunc).
			AddInput(bc.AssetAmount{AssetID: rightA, Amount: 1}, right.PKScript(), nil).
			AddInput(bc.AssetAmount{AssetID: assetID, Amount: tc.amount}, tc.prev.PKScript(), sigscript).
			AddOutput(bc.AssetAmount{AssetID: rightA, Amount: 1}, right.PKScript())
		for tok, amount := range tc.outs {
			tx = tx.AddOutput(bc.AssetAmount{AssetID: assetID, Amount: amount}, tok.PKScript())
		}
		err = tx.Execute(ctx, 1)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestRegisterToVoteClause(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		rightAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		otherRightAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokenAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		rightAssetAmount  = bc.AssetAmount{
			AssetID: rightAssetID,
			Amount:  1,
		}
		right = rightScriptData{
			Delegatable:    true,
			OwnershipChain: bc.Hash{}, // 0x000...000
			HolderScript:   []byte{txscript.OP_1},
		}
	)

	testCases := []struct {
		err           error
		right         *rightScriptData
		amount        uint64
		registrations []Registration
		prev          tokenScriptData
		outs          map[*tokenScriptData]uint64
	}{
		{
			// single-party registration, entire lot
			amount: 200,
			registrations: []Registration{
				{ID: []byte{0xde, 0xad, 0xbe, 0xef}, Amount: 200},
			},
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{
					RegistrationID: []byte{0xde, 0xad, 0xbe, 0xef},
					Right:          rightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered,
				}: 200,
			},
		},
		{
			// single-party registration, no ID, entire lot
			amount: 200,
			registrations: []Registration{
				{ID: nil, Amount: 200},
			},
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{
					RegistrationID: nil,
					Right:          rightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered,
				}: 200,
			},
		},
		{
			// multi-party registration with change.
			amount: 1000,
			registrations: []Registration{
				{ID: []byte{0xde, 0xad, 0xbe, 0xef}, Amount: 200},
				{ID: []byte{0xCA, 0xFE, 0xD0, 0x0D}, Amount: 100},
				{ID: []byte{0xC0, 0x00, 0x10, 0xFF}, Amount: 500},
			},
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{
					RegistrationID: []byte{0xde, 0xad, 0xbe, 0xef},
					Right:          rightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered,
				}: 200,
				&tokenScriptData{
					RegistrationID: []byte{0xca, 0xfe, 0xd0, 0x0d},
					Right:          rightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered,
				}: 100,
				&tokenScriptData{
					RegistrationID: []byte{0xC0, 0x00, 0x10, 0xFF},
					Right:          rightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered,
				}: 500,
				&tokenScriptData{
					Right:       rightAssetID,
					AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
					State:       stateDistributed,
				}: 300,
			},
		},
		{
			// Output has wrong voting right asset id.
			err:    txscript.ErrStackVerifyFailed,
			amount: 200,
			registrations: []Registration{
				{ID: []byte{0xde, 0xad, 0xbe, 0xef}, Amount: 200},
			},
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateDistributed,
				Vote:        2,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{
					RegistrationID: []byte{0xde, 0xad, 0xbe, 0xef},
					Right:          otherRightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered,
					Vote:           2,
				}: 200,
			},
		},
		{
			// Cannot move from FINISHED even if distributed.
			err:    txscript.ErrStackVerifyFailed,
			amount: 200,
			registrations: []Registration{
				{ID: []byte{0xde, 0xad, 0xbe, 0xef}, Amount: 200},
			},
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateDistributed | stateFinished,
				Vote:        2,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{
					RegistrationID: []byte{0xde, 0xad, 0xbe, 0xef},
					Right:          rightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered | stateFinished,
					Vote:           2,
				}: 200,
			},
		},
		{
			// State changed to VOTED, not REGISTERED.
			err:    txscript.ErrStackVerifyFailed,
			amount: 100,
			registrations: []Registration{
				{ID: []byte{0xde, 0xad, 0xbe, 0xef}, Amount: 100},
			},
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateDistributed,
				Vote:        2,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{
					RegistrationID: []byte{0xde, 0xad, 0xbe, 0xef},
					Right:          rightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateVoted,
					Vote:           2,
				}: 100,
			},
		},
		{
			// Tx has a voting right, but it's the wrong voting
			// right.
			err:    txscript.ErrStackVerifyFailed,
			amount: 200,
			registrations: []Registration{
				{ID: []byte{0xde, 0xad, 0xbe, 0xef}, Amount: 200},
			},
			prev: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateDistributed,
			},
			outs: map[*tokenScriptData]uint64{
				&tokenScriptData{
					RegistrationID: []byte{0xde, 0xad, 0xbe, 0xef},
					Right:          otherRightAssetID,
					AdminScript:    []byte{0xde, 0xad, 0xbe, 0xef},
					State:          stateRegistered,
				}: 200,
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		for _, r := range tc.registrations {
			sb = sb.AddInt64(int64(r.Amount)).AddData(r.ID)
		}
		sb = sb.
			AddInt64(int64(len(tc.registrations))).
			AddData(right.PKScript()).
			AddInt64(int64(clauseRegister)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		r := right
		if tc.right != nil {
			r = *tc.right
		}
		tx := txscripttest.NewTestTx(mockTimeFunc).
			AddInput(rightAssetAmount, r.PKScript(), nil).
			AddInput(bc.AssetAmount{AssetID: tokenAssetID, Amount: tc.amount}, tc.prev.PKScript(), sigscript).
			AddOutput(rightAssetAmount, r.PKScript())
		for out, amt := range tc.outs {
			tx = tx.AddOutput(bc.AssetAmount{AssetID: tokenAssetID, Amount: amt}, out.PKScript())
		}
		err = tx.Execute(ctx, 1)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestVoteClause(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		rightAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		otherRightAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokenAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokensAssetAmount = bc.AssetAmount{
			AssetID: tokenAssetID,
			Amount:  200,
		}
		rightAssetAmount = bc.AssetAmount{
			AssetID: rightAssetID,
			Amount:  1,
		}
		right = rightScriptData{
			Delegatable:    true,
			OwnershipChain: bc.Hash{}, // 0x000...000
			HolderScript:   []byte{txscript.OP_1},
		}
	)

	testCases := []struct {
		err   error
		right *rightScriptData
		prev  tokenScriptData
		out   tokenScriptData
	}{
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateRegistered,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateVoted,
				Vote:        2,
			},
		},
		{
			// Already voted, changing vote
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateVoted,
				Vote:        0,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateVoted,
				Vote:        1,
			},
		},
		{
			// Output has wrong voting right asset id.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateRegistered,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateVoted,
				Vote:        2,
			},
		},
		{
			// Cannot move from FINISHED even if REGISTERED.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateRegistered | stateFinished,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateVoted | stateFinished,
				Vote:        2,
			},
		},
		{
			// Tx has a voting right, but it's the wrong voting
			// right.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateRegistered,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateVoted,
				Vote:        2,
			},
		},
		{
			// Voting right output script doesn't match the token holding
			// contract sigscript param.
			err: txscript.ErrStackVerifyFailed,
			right: &rightScriptData{
				HolderScript: []byte{txscript.OP_2},
			},
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateRegistered,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				State:       stateVoted,
				Vote:        2,
			},
		},
	}

	for i, tc := range testCases {
		r := right
		if tc.right != nil {
			r = *tc.right
		}

		sb := txscript.NewScriptBuilder()
		sb = sb.
			AddInt64(tc.out.Vote).
			AddData(r.PKScript()).
			AddInt64(int64(clauseVote)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(rightAssetAmount, right.PKScript(), nil).
			AddInput(tokensAssetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(rightAssetAmount, right.PKScript()).
			AddOutput(tokensAssetAmount, tc.out.PKScript()).
			Execute(ctx, 1)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestFinishVoteClause(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		rightAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		otherRightAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokenAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokensAssetAmount = bc.AssetAmount{
			AssetID: tokenAssetID,
			Amount:  200,
		}
	)

	testCases := []struct {
		err  error
		prev tokenScriptData
		out  tokenScriptData
	}{
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
		},
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        8,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateFinished,
				Vote:        8,
			},
		},
		{
			// Voting token state already finished.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
		},
		{
			// Voting token state invalid.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
		},
		{
			// Admin script does not authorize.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1, txscript.OP_DROP, txscript.OP_0},
				State:       stateDistributed,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1, txscript.OP_DROP, txscript.OP_0},
				State:       stateDistributed + stateFinished,
			},
		},
		{
			// Output has wrong voting right asset id.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateFinished,
				Vote:        2,
			},
		},
		{
			// Cannot change the base state at the same time.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateRegistered | stateFinished,
				Vote:        2,
			},
		},
		{
			// Vote changed during closing
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateFinished,
				Vote:        3,
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		sb = sb.
			AddInt64(int64(clauseFinish)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(tokensAssetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(tokensAssetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestRetireClause(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		rightAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokenAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokensAssetAmount = bc.AssetAmount{
			AssetID: tokenAssetID,
			Amount:  200,
		}
	)

	testCases := []struct {
		err          error
		prev         tokenScriptData
		outputScript []byte
	}{
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
			outputScript: []byte{txscript.OP_RETURN},
		},
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateFinished,
				Vote:        2,
			},
			outputScript: []byte{txscript.OP_RETURN},
		},
		{
			// Output sends back into the voting token contract.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
				Vote:        2,
			},
			outputScript: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
				Vote:        1,
			}.PKScript(),
		},
		{
			// Token must be FINISHED to be retired.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        2,
			},
			outputScript: []byte{txscript.OP_RETURN},
		},
		{
			// Admin script must authorize token retirement.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1, txscript.OP_DROP, txscript.OP_0},
				State:       stateVoted | stateFinished,
				Vote:        2,
			},
			outputScript: []byte{txscript.OP_RETURN},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		sb = sb.
			AddInt64(int64(clauseRetire)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(tokensAssetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(tokensAssetAmount, tc.outputScript).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestInvalidateVoteClause(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		rightAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		otherRightAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokenAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokensAssetAmount = bc.AssetAmount{
			AssetID: tokenAssetID,
			Amount:  200,
		}
	)

	testCases := []struct {
		err  error
		prev tokenScriptData
		out  tokenScriptData
	}{
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
			},
		},
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        8,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateInvalid,
				Vote:        8,
			},
		},
		{
			// Voting token state already invalid.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
			},
		},
		{
			// Voting token state finished.
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
			},
		},
		{
			// Admin script does not authorize.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1, txscript.OP_DROP, txscript.OP_0},
				State:       stateDistributed,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1, txscript.OP_DROP, txscript.OP_0},
				State:       stateDistributed | stateInvalid,
			},
		},
		{
			// Output has wrong voting right asset id.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateInvalid,
				Vote:        2,
			},
		},
		{
			// Cannot change the base state at the same time.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateRegistered | stateInvalid,
				Vote:        2,
			},
		},
		{
			// Vote changed during closing
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateInvalid,
				Vote:        3,
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		sb = sb.
			AddInt64(int64(clauseInvalidate)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(tokensAssetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(tokensAssetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

// TestTokenContractValidMatch tests generating a pkscript from a voting token.
// The generated pkscript is then used in the voting token p2c detection
// flow, where it should be found to match the contract. Then the decoded
// voting token and the original voting token are checked for equality.
func TestTokenContractValidMatch(t *testing.T) {
	testCases := []tokenScriptData{
		{
			Right:       bc.AssetID{},
			AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
			State:       stateRegistered,
			Vote:        5,
		},
		{
			Right:       bc.AssetID{0x01},
			AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
			State:       stateVoted | stateFinished,
			Vote:        1,
		},
	}

	for i, want := range testCases {
		script := want.PKScript()
		got, err := testTokenContract(script)
		if err != nil {
			t.Errorf("%d: testing voting token contract for %x: %s", i, script, err)
			continue
		}

		if got == nil {
			t.Errorf("%d: No match for pkscript %x generated from %#v", i, script, want)
			continue
		}

		if got.Right != want.Right {
			t.Errorf("%d: token.Right, got=%#v want=%#v", i, got.Right, want.Right)
		}
		if !bytes.Equal(got.AdminScript, want.AdminScript) {
			t.Errorf("%d: token.AdminScript, got=%#v want=%#v", i, got.AdminScript, want.AdminScript)
		}
		if got.Vote != want.Vote {
			t.Errorf("%d: token.Vote, got=%#v want=%#v", i, got.Vote, want.Vote)
		}
	}
}

func TestResetClause(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		rightAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		otherRightAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokenAssetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		tokensAssetAmount = bc.AssetAmount{
			AssetID: tokenAssetID,
			Amount:  200,
		}
	)

	testCases := []struct {
		err  error
		prev tokenScriptData
		out  tokenScriptData
	}{
		{
			// Move from voted to registered
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        8,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateRegistered,
			},
		},
		{
			// Cannot reset once finished.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
				Vote:        1,
			},
		},
		{
			// Cannot reset if invalid.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateInvalid,
				Vote:        1,
			},
		},
		{
			// Cannot reset to voted.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed | stateFinished,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateFinished,
				Vote:        1,
			},
		},
		{
			// Admin script does not authorize.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1, txscript.OP_DROP, txscript.OP_0},
				State:       stateDistributed,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1, txscript.OP_DROP, txscript.OP_0},
				State:       stateDistributed + stateFinished,
			},
		},
		{
			// Output has wrong voting right asset id.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateFinished,
				Vote:        2,
			},
		},
		{
			// Cannot change the base state at the same time.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateDistributed,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateRegistered | stateFinished,
				Vote:        2,
			},
		},
		{
			// Vote changed during closing
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				State:       stateVoted | stateFinished,
				Vote:        3,
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		sb = sb.
			AddInt64(int64(tc.out.State)).
			AddInt64(int64(clauseReset)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(tokensAssetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(tokensAssetAmount, tc.out.PKScript()).
			Execute(ctx, 0)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

// TestTokenContractInvalidScript tests that testTokenContract correctly
// fails on pkscripts that are paid to the token contract but are
// improperly formatted.
func TestTokenContractInvalidMatch(t *testing.T) {
	testCaseScriptParams := [][]txscript.Item{
		{ // no parameters
		},
		{ // not enough parameters
			txscript.DataItem(nil), txscript.DataItem(nil), txscript.DataItem(nil),
		},
		{ // enough parameters, but all empty
			txscript.DataItem(nil), txscript.DataItem(nil), txscript.DataItem(nil), txscript.DataItem(nil),
		},
		{ // asset id not long enough
			txscript.DataItem([]byte{0x01}),                   // voting right asset id = 0x01
			txscript.DataItem([]byte{0xde, 0xad, 0xbe, 0xef}), // admin script = 0xdeadbeef
			txscript.DataItem([]byte{byte(clauseRegister)}),   // state = REGISTERED
			txscript.NumItem(int64(1)),                        // vote = 1
		},
		{ // too many parameters
			txscript.DataItem(exampleHash[:]),                 // voting right asset id = example hash
			txscript.DataItem([]byte{0xde, 0xad, 0xbe, 0xef}), // admin script = 0xdeadbeef
			txscript.DataItem([]byte{byte(clauseRegister)}),   // state = REGISTERED
			txscript.NumItem(int64(1)),                        // vote = 1
			txscript.DataItem([]byte{0x00, 0x01}),             // garbage parameter
		},
	}

	for i, params := range testCaseScriptParams {
		script, err := txscript.PayToContractHash(tokenHoldingContractHash, params, scriptVersion)
		if err != nil {
			t.Errorf("%d: building pkscript: %s", i, err)
			continue
		}

		data, err := testTokenContract(script)
		if err != nil {
			t.Errorf("%d: testing token contract for %x: %s", i, script, err)
			continue
		}
		if data != nil {
			t.Errorf("%d: Match for pkscript %x generated from params: %#v", i, script, params)
			continue
		}
	}
}
