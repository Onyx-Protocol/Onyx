package voting

import (
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/hash256"
)

type TokenState byte

func (ts TokenState) Finished() bool    { return ts&stateFinished == stateFinished }
func (ts TokenState) Base() TokenState  { return 0x0F & ts }
func (ts TokenState) Distributed() bool { return ts.Base() == stateDistributed }
func (ts TokenState) Registered() bool  { return ts.Base() == stateRegistered }
func (ts TokenState) Voted() bool       { return ts.Base() == stateVoted }
func (ts TokenState) String() string {
	switch ts.Base() {
	case stateDistributed:
		return "distributed"
	case stateRegistered:
		return "registered"
	case stateVoted:
		return "voted"
	}
	return ""
}

const (
	stateDistributed TokenState = 0x00
	stateRegistered             = 0x01
	stateVoted                  = 0x02
	stateFinished               = 0x10 // bit mask
)

type tokenContractClause int64

const (
	clauseRegister tokenContractClause = 1
	clauseVote                         = 2
	clauseFinish                       = 3
	clauseReset                        = 4
	clauseRetire                       = 5
)

// tokenScriptData encapsulates all the data stored within the p2c script
// for the voting token holding contract.
type tokenScriptData struct {
	Right       bc.AssetID
	AdminScript []byte
	State       TokenState
	Vote        int64
}

// PKScript constructs a script address to pay into the holding
// contract for this voting token. It implements the txbuilder.Receiver
// interface.
func (t tokenScriptData) PKScript() []byte {
	var (
		params [][]byte
	)

	params = append(params, txscript.Int64ToScriptBytes(t.Vote))
	params = append(params, []byte{byte(t.State)})
	params = append(params, t.AdminScript)
	params = append(params, t.Right[:])

	script, err := txscript.PayToContractHash(tokenHoldingContractHash, params, scriptVersion)
	if err != nil {
		return nil
	}
	return script
}

// testTokenContract tests whether the given pkscript is a voting
// token holding contract.
func testTokenContract(pkscript []byte) (*tokenScriptData, error) {
	parsedScriptVersion, _, _, params := txscript.ParseP2C(pkscript, tokenHoldingContract)
	if parsedScriptVersion == nil {
		return nil, nil
	}
	if len(params) != 4 {
		return nil, nil
	}

	var (
		err   error
		token tokenScriptData
	)

	// Corresponding voting right's asset ID.
	if cap(token.Right) != len(params[3]) {
		return nil, nil
	}
	copy(token.Right[:], params[3])

	// Voting system administrator script
	token.AdminScript = make([]byte, len(params[2]))
	copy(token.AdminScript, params[2])

	// The current state of the token.
	var state int64
	state, err = txscript.AsInt64(params[1])
	if err != nil {
		return nil, err
	}
	token.State = TokenState(state)

	// The currently selected option, if any.
	token.Vote, err = txscript.AsInt64(params[0])
	if err != nil {
		return nil, err
	}
	return &token, nil
}

const (
	// tokenHoldingContractString contains the entire voting token holding
	// contract script. For now, it's structured as a series of IF...ENDIF
	// clauses. In the future, we will use merkleized scripts, as documented
	// in the fedchain p2c documentation.
	//
	// This script with documentation and comments is available here:
	// https://gist.github.com/jbowens/ae16b535c856c137830e
	//
	// 1 - Register to vote
	// 2 - Vote
	// 3 - Finish
	// 4 - Reset
	// 5 - Retire
	tokenHoldingContractString = `
		4 ROLL
		DUP 1 EQUAL IF
			DROP SWAP
			0 NUMEQUALVERIFY
			OP_1 3 PICK
			5 ROLL FINDOUTPUT VERIFY
			DATA_2 0x5275
			3 ROLL CATPUSHDATA
			ROT CATPUSHDATA
			1 CATPUSHDATA
			SWAP CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT AMOUNT ASSET ROT
			RESERVEOUTPUT
		ENDIF
		DUP 2 EQUAL IF
			2DROP
			4 ROLL
			SWAP DUP 1 EQUAL SWAP 2 EQUAL BOOLOR VERIFY
			1 3 PICK
			5 ROLL FINDOUTPUT VERIFY
			DATA_2 0x5275
			3 ROLL CATPUSHDATA
			ROT CATPUSHDATA
			2 CATPUSHDATA
			SWAP CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT AMOUNT ASSET ROT
			RESERVEOUTPUT
		ENDIF
		DUP 3 EQUAL IF
			DROP
			DATA_2 0x5275
			4 ROLL CATPUSHDATA
			3 PICK CATPUSHDATA
			ROT
			DUP 16 LESSTHAN VERIFY
			16 ADD CATPUSHDATA
			SWAP CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT AMOUNT ASSET ROT
			RESERVEOUTPUT VERIFY
			EVAL
		ENDIF
		DUP 4 EQUAL IF
			2DROP DROP
			DATA_2 0x5275
			ROT CATPUSHDATA
			OVER CATPUSHDATA
			ROT CATPUSHDATA
			0 CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT AMOUNT ASSET ROT
			RESERVEOUTPUT VERIFY
			EVAL
		ENDIF
		DUP 5 EQUAL IF
			2DROP
			16 GREATERTHANOREQUAL VERIFY
			NIP
			AMOUNT ASSET DATA_1 0x6a
			RESERVEOUTPUT VERIFY
			EVAL
		ENDIF
	`
)

var (
	tokenHoldingContract     []byte
	tokenHoldingContractHash [hash256.Size]byte
)

func init() {
	var err error
	tokenHoldingContract, err = txscript.ParseScriptString(tokenHoldingContractString)
	if err != nil {
		panic("failed parsing voting token holding script: " + err.Error())
	}
	// TODO(jackson): Before going to production, we'll probably want to hard-code the
	// contract hash and panic if the contract changes.
	tokenHoldingContractHash = hash256.Sum(tokenHoldingContract)
}
