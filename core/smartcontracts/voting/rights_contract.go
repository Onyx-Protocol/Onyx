package voting

import (
	"bytes"
	"fmt"

	"golang.org/x/crypto/sha3"

	"chain/cos/bc"
	"chain/cos/txscript"
)

// scriptVersion encodes the version of the scripting language required
// for executing the voting rights contract.
var scriptVersion = txscript.ScriptVersion2

const (
	// pinnedRightsContractHash stores the hash of the voting rights contract.
	// Changes to the the contract will require updating the hash.
	pinnedRightsContractHash = "292d0b53628f4be2e2441de50863474c124a9e9dafc2574b1903586cbb0d9bf3"
)

type rightsContractClause int64

const (
	clauseAuthenticate rightsContractClause = 1
	clauseTransfer                          = 2
	clauseDelegate                          = 3
	clauseRecall                            = 4
	clauseOverride                          = 5
	clauseCancel                            = 6
)

// rightScriptData encapsulates all the data stored within the p2c script
// for the voting rights holding contract.
type rightScriptData struct {
	AdminScript    []byte
	HolderScript   []byte
	OwnershipChain bc.Hash
	Delegatable    bool
}

// PKScript constructs a script address to pay into the holding
// contract for this voting right. It implements the txbuilder.Receiver
// interface.
func (r rightScriptData) PKScript() []byte {
	params := make([]txscript.Item, 0, 4)

	params = append(params, txscript.BoolItem(r.Delegatable))
	params = append(params, txscript.DataItem(r.OwnershipChain[:]))
	params = append(params, txscript.DataItem(r.HolderScript))
	params = append(params, txscript.DataItem(r.AdminScript))

	script, err := txscript.PayToContractHash(rightsHoldingContractHash, params, scriptVersion)
	if err != nil {
		return nil
	}
	return script
}

// testRightsContract tests whether the given pkscript is a voting
// rights holding contract.
func testRightsContract(pkscript []byte) (*rightScriptData, error) {
	parsedScriptVersion, _, _, params := txscript.ParseP2C(pkscript, rightsHoldingContract)
	if parsedScriptVersion == nil {
		return nil, nil
	}
	if len(params) != 4 {
		return nil, nil
	}

	var right rightScriptData

	// delegatable bool
	right.Delegatable = txscript.AsBool(params[0])

	// chain of ownership hash
	if cap(right.OwnershipChain) != len(params[1]) {
		return nil, nil
	}
	copy(right.OwnershipChain[:], params[1])

	// script identifying holder of the right
	right.HolderScript = make([]byte, len(params[2]))
	copy(right.HolderScript, params[2])

	// script identifying the admin of the system
	right.AdminScript = make([]byte, len(params[3]))
	copy(right.AdminScript, params[3])

	return &right, nil
}

// testRightsSigscript tests whether the given sigscript is redeeming a
// voting rights holding contract. It will return the clause being used,
// and a slice of the other clause parameters.
func testRightsSigscript(sigscript []byte) (ok bool, c rightsContractClause, params [][]byte) {
	data, err := txscript.PushedData(sigscript)
	if err != nil {
		return false, c, nil
	}
	if len(data) < 2 {
		return false, c, nil
	}
	if !bytes.Equal(data[len(data)-1], rightsHoldingContract) {
		return false, c, nil
	}

	clauseBytes := data[len(data)-2]
	if len(clauseBytes) != 1 {
		return false, c, nil
	}
	c = rightsContractClause(clauseBytes[0])
	if c < clauseAuthenticate || c > clauseCancel {
		return false, c, nil
	}
	return true, c, data[:len(data)-2]
}

func paramsPopInt64(params [][]byte, valid *bool) ([][]byte, int64) {
	if len(params) < 1 {
		*valid = false
		return params, 0
	}
	v, err := txscript.AsInt64(params[len(params)-1])
	*valid = *valid && err == nil
	return params[:len(params)-1], v
}

func paramsPopBool(params [][]byte, valid *bool) ([][]byte, bool) {
	if len(params) < 1 {
		*valid = false
		return params, false
	}
	return params[:len(params)-1], txscript.AsBool(params[len(params)-1])
}

func paramsPopBytes(params [][]byte, valid *bool) ([][]byte, []byte) {
	if len(params) < 1 {
		*valid = false
		return params, nil
	}
	return params[:len(params)-1], params[len(params)-1]
}

func paramsPopHash(params [][]byte, valid *bool) ([][]byte, bc.Hash) {
	if len(params) < 1 {
		*valid = false
		return nil, bc.Hash{}
	}
	var hash bc.Hash
	copy(hash[:], params[len(params)-1])
	return params[:len(params)-1], hash
}

const (
	// rightsHoldingContractString contains the entire rights holding
	// contract script. For now, it's structured as a series of IF...ENDIF
	// clauses. In the future, we will use merkleized scripts, as documented in
	// the Chain OS p2c documentation.
	//
	// This script with documentation and comments is available here:
	// https://gist.github.com/erykwalder/ea68d529631731e6586685869e7bb747
	//
	// 1 - Authenticate
	// 2 - Transfer
	// 3 - Delegate
	// 4 - Recall
	// 5 - Override
	// 6 - Cancel       (Unimplemented)
	rightsHoldingContractString = `
		4 ROLL
		DUP 1 EQUAL IF
			DROP 2DROP
			AMOUNT ASSET OUTPUTSCRIPT
			RESERVEOUTPUT VERIFY
			NIP
			EVAL
		ENDIF
		DUP 2 EQUAL IF
			DROP
			DATA_2 0x5275
			4 PICK CATPUSHDATA
			5 ROLL CATPUSHDATA
			2 ROLL CATPUSHDATA
			SWAP CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT
			AMOUNT ASSET 2 ROLL
			RESERVEOUTPUT VERIFY
			NIP
			EVAL
		ENDIF
		DUP 3 EQUAL IF
			DROP
			VERIFY
			1 PICK SHA3
			SWAP CAT SHA3
			DATA_2 0x5275
			3 PICK CATPUSHDATA
			4 ROLL CATPUSHDATA
			SWAP CATPUSHDATA
			3 ROLL CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT
			AMOUNT ASSET ROT
			RESERVEOUTPUT VERIFY
			NIP
			EVAL
		ENDIF
		DUP 4 EQUAL IF
			DROP
			4 ROLL SIZE
			DATA_1 0x20 EQUALVERIFY
			5 PICK SHA3
			1 PICK CAT SHA3
			7 ROLL
			WHILE
				8 ROLL
				ROT CAT SHA3
				SWAP 1SUB
			ENDWHILE
			3 ROLL EQUALVERIFY
			DATA_2 0x5275
			4 PICK CATPUSHDATA
			5 PICK CATPUSHDATA
			SWAP CATPUSHDATA
			1 CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT
			AMOUNT ASSET ROT
			RESERVEOUTPUT VERIFY
			2DROP DROP
			EVAL
		ENDIF
	DUP 5 EQUAL IF
		DROP
		7 PICK
		6 PICK
		8 ROLL
		WHILE
			SWAP 9 ROLL SWAP
			CAT SHA3
			SWAP 1SUB
		ENDWHILE
		3 PICK EQUALVERIFY
		6 PICK NOTIF
			8 PICK
			3 PICK NOTIF
				4 ROLL EQUALVERIFY
				DROP
			ELSE
				SHA3
				EQUALVERIFY
				ROT DROP
			ENDIF
		ELSE
			DROP ROT DROP
		ENDIF
		5 ROLL
		1SUB DUP 6 ADD ROLL
		6 ROLL
		2 ROLL
		WHILE
			7 ROLL SHA3
			ROT CAT SHA3
			SWAP 1SUB
		ENDWHILE
		DATA_2 0x5275
		5 PICK CATPUSHDATA
		ROT CATPUSHDATA
		SWAP CATPUSHDATA
		4 ROLL CATPUSHDATA
		OUTPUTSCRIPT
		DATA_1 0x27 RIGHT CAT
		AMOUNT ASSET ROT
		RESERVEOUTPUT VERIFY
		2DROP EVAL
	ENDIF
	`
)

var (
	rightsHoldingContract     []byte
	rightsHoldingContractHash [32]byte
)

func init() {
	var err error
	rightsHoldingContract, err = txscript.ParseScriptString(rightsHoldingContractString)
	if err != nil {
		panic("failed parsing voting rights holding script: " + err.Error())
	}
	rightsHoldingContractHash = sha3.Sum256(rightsHoldingContract)

	if pinnedRightsContractHash != bc.Hash(rightsHoldingContractHash).String() {
		panic(fmt.Sprintf("Expected right contract hash %s, current contract has hash %x",
			pinnedRightsContractHash, rightsHoldingContractHash[:]))
	}
}

// calculateOwnershipChain extends the provided chain of ownership with the provided
// holder using the formula:
//
//     Sha3(Sha3(holder) + oldchain)
//
func calculateOwnershipChain(oldChain bc.Hash, holder []byte) bc.Hash {
	hash := sha3.Sum256(holder)
	data := append(hash[:], oldChain[:]...)
	return sha3.Sum256(data)
}
