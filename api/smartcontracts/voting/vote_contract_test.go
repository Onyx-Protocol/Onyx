package voting

import (
	"bytes"
	"reflect"
	"testing"

	"chain/api/asset/assettest"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/cos/txscript/txscripttest"
	"chain/crypto/hash256"
	"chain/database/pg/pgtest"
)

func TestIntendToVoteClause(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

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
			Deadline:       infiniteDeadline,
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
				OptionCount: 3,
				State:       stateDistributed,
				SecretHash:  exampleHash,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  exampleHash,
			},
		},
		{
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: exampleHash[:],
				OptionCount: 10,
				State:       stateDistributed,
				SecretHash:  exampleHash,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: exampleHash[:],
				OptionCount: 10,
				State:       stateIntended,
				SecretHash:  exampleHash,
			},
		},
		{
			// Output has wrong voting right asset id.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateDistributed,
				SecretHash:  exampleHash,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  exampleHash,
				Vote:        2,
			},
		},
		{
			// Cannot move from FINISHED even if distributed.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateDistributed | stateFinished,
				SecretHash:  exampleHash,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended | stateFinished,
				SecretHash:  exampleHash,
				Vote:        2,
			},
		},
		{
			// State changed to VOTED, not INTENDED.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateDistributed,
				SecretHash:  exampleHash,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted,
				SecretHash:  exampleHash,
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
				OptionCount: 3,
				State:       stateDistributed,
				SecretHash:  exampleHash,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  exampleHash,
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
				OptionCount: 3,
				State:       stateDistributed,
				SecretHash:  exampleHash,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  exampleHash,
				Vote:        2,
			},
		},
	}

	for i, tc := range testCases {
		sb := txscript.NewScriptBuilder()
		sb = sb.
			AddData(right.PKScript()).
			AddInt64(int64(clauseIntendToVote)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		r := right
		if tc.right != nil {
			r = *tc.right
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(rightAssetAmount, r.PKScript(), nil).
			AddInput(tokensAssetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(rightAssetAmount, r.PKScript()).
			AddOutput(tokensAssetAmount, tc.out.PKScript()).
			Execute(ctx, 1)
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: got=%s want=%s", i, err, tc.err)
		}
	}
}

func TestVoteClause(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

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
			Deadline:       infiniteDeadline,
			Delegatable:    true,
			OwnershipChain: bc.Hash{}, // 0x000...000
			HolderScript:   []byte{txscript.OP_1},
		}
		votingSecret     = []byte("an example voting secret")
		votingSecretHash = hash256.Sum(votingSecret)
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
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  votingSecretHash,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted,
				SecretHash:  votingSecretHash,
				Vote:        2,
			},
		},
		{
			// Already voted, changing vote
			err: nil,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 2,
				State:       stateVoted,
				SecretHash:  votingSecretHash,
				Vote:        1,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 2,
				State:       stateVoted,
				SecretHash:  votingSecretHash,
				Vote:        2,
			},
		},
		{
			// Wrong voting secret
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  exampleHash,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted,
				SecretHash:  exampleHash,
				Vote:        2,
			},
		},
		{
			// Output has wrong voting right asset id.
			err: txscript.ErrStackScriptFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  votingSecretHash,
				Vote:        2,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted,
				SecretHash:  votingSecretHash,
				Vote:        2,
			},
		},
		{
			// Cannot move from FINISHED even if INTENDED.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended | stateFinished,
				SecretHash:  votingSecretHash,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted | stateFinished,
				SecretHash:  votingSecretHash,
				Vote:        2,
			},
		},
		{
			// Vote is outside the range of options.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  votingSecretHash,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted,
				SecretHash:  votingSecretHash,
				Vote:        99999,
			},
		},
		{
			// Tx has a voting right, but it's the wrong voting
			// right.
			err: txscript.ErrStackVerifyFailed,
			prev: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  votingSecretHash,
			},
			out: tokenScriptData{
				Right:       otherRightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted,
				SecretHash:  votingSecretHash,
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
				OptionCount: 3,
				State:       stateIntended,
				SecretHash:  exampleHash,
			},
			out: tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
				OptionCount: 3,
				State:       stateVoted,
				SecretHash:  exampleHash,
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
			AddData(votingSecret).
			AddData(r.PKScript()).
			AddInt64(int64(clauseVote)).
			AddData(tokenHoldingContract)
		sigscript, err := sb.Script()
		if err != nil {
			t.Fatal(err)
		}
		err = txscripttest.NewTestTx(mockTimeFunc).
			AddInput(rightAssetAmount, r.PKScript(), nil).
			AddInput(tokensAssetAmount, tc.prev.PKScript(), sigscript).
			AddOutput(rightAssetAmount, r.PKScript()).
			AddOutput(tokensAssetAmount, tc.out.PKScript()).
			Execute(ctx, 1)
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
			OptionCount: 10,
			State:       stateIntended,
			SecretHash:  exampleHash,
			Vote:        5,
		},
		{
			Right:       bc.AssetID{0x01},
			AdminScript: []byte{0xde, 0xad, 0xbe, 0xef},
			OptionCount: 2,
			State:       stateVoted | stateFinished,
			SecretHash:  bc.Hash{},
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
		if got.OptionCount != want.OptionCount {
			t.Errorf("%d: token.OptionCount, got=%#v want=%#v", i, got.OptionCount, want.OptionCount)
		}
		if got.SecretHash != want.SecretHash {
			t.Errorf("%d: token.SecretHash, got=%#v want=%#v", i, got.SecretHash, want.SecretHash)
		}
		if got.Vote != want.Vote {
			t.Errorf("%d: token.Vote, got=%#v want=%#v", i, got.Vote, want.Vote)
		}
	}
}

// TestTokenContractInvalidScript tests that testTokenContract correctly
// fails on pkscripts that are paid to the token contract but are
// improperly formatted.
func TestTokenContractInvalidMatch(t *testing.T) {
	testCaseScriptParams := [][][]byte{
		{ // no parameters
		},
		{ // not enough parameters
			[]byte{}, []byte{}, []byte{},
		},
		{ // enough parameters, but all empty
			[]byte{}, []byte{}, []byte{}, []byte{}, []byte{}, []byte{},
		},
		{ // asset id not long enough
			[]byte{0x01},                     // voting right asset id = 0x01
			[]byte{0xde, 0xad, 0xbe, 0xef},   // admin script = 0xdeadbeef
			[]byte{0x52},                     // option count = 2
			[]byte{byte(clauseIntendToVote)}, // state = INTEND_TO_VOTE
			exampleHash[:],                   // secret hash = example hash
			[]byte{0x51},                     // vote = 1
		},
		{ // too many parameters
			exampleHash[:],                   // voting right asset id = example hash
			[]byte{0xde, 0xad, 0xbe, 0xef},   // admin script = 0xdeadbeef
			[]byte{0x52},                     // option count = 2
			[]byte{byte(clauseIntendToVote)}, // state = INTEND_TO_VOTE
			exampleHash[:],                   // secret hash = example hash
			[]byte{0x51},                     // vote = 1
			[]byte{0x00, 0x01},               // garbage parameter
		},
	}

	for i, params := range testCaseScriptParams {
		addr := txscript.NewAddressContractHash(tokenHoldingContractHash[:], scriptVersion, params)
		script := addr.ScriptAddress()

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
