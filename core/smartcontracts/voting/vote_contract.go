package voting

import (
	"bytes"
	"fmt"

	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/hash256"
)

const (
	// pinnedTokenContractHash stores the hash of the voting token contract.
	// Changes to the contract will require updating the hash.
	pinnedTokenContractHash = "7659a7102446ace00ca187998852db6c37c8e96ae5a21f77eb36efa2673d3e11"
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
	clauseRedistribute tokenContractClause = 1
	clauseRegister                         = 2
	clauseVote                             = 3
	clauseFinish                           = 4
	clauseReset                            = 5
	clauseRetire                           = 6
)

// tokenScriptData encapsulates all the data stored within the p2c script
// for the voting token holding contract.
type tokenScriptData struct {
	RegistrationID []byte
	Right          bc.AssetID
	AdminScript    []byte
	State          TokenState
	Vote           int64
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
	params = append(params, t.RegistrationID)

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
	if len(params) != 5 {
		return nil, nil
	}

	var (
		err   error
		token tokenScriptData
	)

	// Registration ID
	token.RegistrationID = make([]byte, len(params[4]))
	copy(token.RegistrationID, params[4])

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

// testTokensSigscript tests whether the given sigscript is redeeming a
// voting token holding contract. It will return the clause being used,
// and a slice of the other clause parameters.
func testTokensSigscript(sigscript []byte) (ok bool, c tokenContractClause, params [][]byte) {
	data, err := txscript.PushedData(sigscript)
	if err != nil {
		return false, c, nil
	}
	if len(data) < 2 {
		return false, c, nil
	}
	if !bytes.Equal(data[len(data)-1], tokenHoldingContract) {
		return false, c, nil
	}

	clauseBytes := data[len(data)-2]
	if len(clauseBytes) != 1 {
		return false, c, nil
	}
	c = tokenContractClause(clauseBytes[0])
	if c < clauseRedistribute || c > clauseRetire {
		return false, c, nil
	}
	return true, c, data[:len(data)-2]
}

type Registration struct {
	ID     []byte
	Amount uint64
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
	// 1 - Redistribute
	// 2 - Register to vote
	// 3 - Vote
	// 4 - Finish
	// 5 - Reset
	// 6 - Retire
	tokenHoldingContractString = `
		5 ROLL
		DUP 1 EQUAL IF
			DROP
			OVER 0 EQUALVERIFY
			1 4 PICK 7 ROLL
			FINDOUTPUT VERIFY
			AMOUNT
			6 ROLL
			WHILE
				DATA_2 0x5275
				7 PICK CATPUSHDATA
				8 ROLL CATPUSHDATA
				5 PICK CATPUSHDATA
				0x00 CATPUSHDATA
				0x00 CATPUSHDATA
				OUTPUTSCRIPT
				DATA_1 0x27 RIGHT CAT
				8 PICK ASSET
				ROT RESERVEOUTPUT VERIFY
				SWAP 7 ROLL SUB
				SWAP 1SUB
			ENDWHILE
			DATA_2 0x5275
			6 ROLL CATPUSHDATA
			5 ROLL CATPUSHDATA
			4 ROLL CATPUSHDATA
			3 ROLL CATPUSHDATA
			ROT CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT CAT
			ASSET SWAP RESERVEOUTPUT
		ENDIF
		DUP 2 EQUAL IF
			DROP
			OVER 0 EQUALVERIFY
			1 4 PICK 7 ROLL
			FINDOUTPUT VERIFY
			AMOUNT
			6 ROLL
			WHILE
				DATA_2 0x5275
				8 ROLL CATPUSHDATA
				6 PICK CATPUSHDATA
				5 PICK CATPUSHDATA
				1 CATPUSHDATA
				0 CATPUSHDATA
				OUTPUTSCRIPT
				DATA_1 0x27 RIGHT CAT
				8 PICK ASSET
				ROT RESERVEOUTPUT VERIFY
				SWAP 7 ROLL SUB
				SWAP 1SUB
			ENDWHILE
			DATA_2 0x5275
			6 ROLL CATPUSHDATA
			5 ROLL CATPUSHDATA
			4 ROLL CATPUSHDATA
			3 ROLL CATPUSHDATA
			ROT CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT CAT
			ASSET SWAP RESERVEOUTPUT
		ENDIF
		DUP 3 EQUAL IF
			2DROP
			5 ROLL
			SWAP DUP 1 EQUAL SWAP 2 EQUAL BOOLOR VERIFY
			1 3 PICK
			6 ROLL FINDOUTPUT VERIFY
			DATA_2 0x5275
			4 ROLL CATPUSHDATA
			3 ROLL CATPUSHDATA
			ROT CATPUSHDATA
			2 CATPUSHDATA
			SWAP CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT AMOUNT ASSET ROT
			RESERVEOUTPUT
		ENDIF
		DUP 4 EQUAL IF
			DROP
			DATA_2 0x5275
			5 ROLL CATPUSHDATA
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
		DUP 5 EQUAL IF
			2DROP DROP
			DATA_2 0x5275
			3 ROLL
			4 PICK NOTIF
				DROP 0
			ENDIF
			CATPUSHDATA
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
		DUP 6 EQUAL IF
			2DROP
			16 GREATERTHANOREQUAL VERIFY
			NIP NIP
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
	tokenHoldingContractHash = hash256.Sum(tokenHoldingContract)

	if pinnedTokenContractHash != bc.Hash(tokenHoldingContractHash).String() {
		panic(fmt.Sprintf("Expected token contract hash %s, current contract has hash %x",
			pinnedTokenContractHash, tokenHoldingContractHash))
	}
}
