package voting

import (
	"bytes"
	"fmt"
	"math"

	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/hash256"
)

// scriptVersion encodes the version of the scripting language required
// for executing the voting rights contract.
var scriptVersion = txscript.ScriptVersion2

const (
	// InfiniteDeadline is a sentinel deadline value to denote infinity.
	// It's value is the max int64 so that the contract byte code can
	// perform ordinary <= comparisons.
	InfiniteDeadline = math.MaxInt64

	// pinnedRightsContractHash stores the hash of the voting rights contract.
	// Changes to the the contract will require updating the hash.
	pinnedRightsContractHash = "d27b4dc74b1f383b12cc522f2ee92cc31f63d33f7280810ef31772450f1f3659"
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
	Deadline       int64
	Delegatable    bool
}

// PKScript constructs a script address to pay into the holding
// contract for this voting right. It implements the txbuilder.Receiver
// interface.
func (r rightScriptData) PKScript() []byte {
	params := make([]txscript.Item, 0, 5)

	params = append(params, txscript.BoolItem(r.Delegatable))
	params = append(params, txscript.NumItem(r.Deadline))
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
	if len(params) != 5 {
		return nil, nil
	}

	var (
		err   error
		right rightScriptData
	)

	// delegatable bool
	right.Delegatable = txscript.AsBool(params[0])

	// deadline in unix secs
	right.Deadline, err = txscript.AsInt64(params[1])
	if err != nil {
		return nil, err
	}

	// chain of ownership hash
	if cap(right.OwnershipChain) != len(params[2]) {
		return nil, nil
	}
	copy(right.OwnershipChain[:], params[2])

	// script identifying holder of the right
	right.HolderScript = make([]byte, len(params[3]))
	copy(right.HolderScript, params[3])

	// script identifying the admin of the system
	right.AdminScript = make([]byte, len(params[4]))
	copy(right.AdminScript, params[4])

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
	// https://gist.github.com/jbowens/ae16b535c856c137830e
	//
	// 1 - Authenticate
	// 2 - Transfer
	// 3 - Delegate
	// 4 - Recall
	// 5 - Override
	// 6 - Cancel       (Unimplemented)
	rightsHoldingContractString = `
		5 ROLL
		DUP 1 EQUAL IF
			DROP
			SWAP
			TIME
			GREATERTHAN VERIFY
			2DROP
			AMOUNT ASSET OUTPUTSCRIPT
			RESERVEOUTPUT VERIFY
			NIP
			EVAL
		ENDIF
		DUP 2 EQUAL IF
			DROP
			1 PICK
			TIME
			GREATERTHAN VERIFY
			DATA_2 0x5275
			5 PICK CATPUSHDATA
			6 ROLL CATPUSHDATA
			3 ROLL CATPUSHDATA
			2 ROLL CATPUSHDATA
			SWAP CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT
			AMOUNT ASSET 2 ROLL
			RESERVEOUTPUT VERIFY
			NIP EVAL
		ENDIF
		DUP 3 EQUAL IF
			DROP
			VERIFY
			DUP TIME
			GREATERTHAN VERIFY
			DUP
			7 PICK
			GREATERTHANOREQUAL VERIFY
			HASH256
			2 PICK HASH256
			SWAP CAT HASH256
			SWAP CAT HASH256
			DATA_2 0x5275
			3 PICK CATPUSHDATA
			4 ROLL CATPUSHDATA
			SWAP CATPUSHDATA
			4 ROLL CATPUSHDATA
			3 ROLL CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT
			AMOUNT ASSET ROT
			RESERVEOUTPUT VERIFY
			NIP EVAL
		ENDIF
		DUP 4 EQUAL IF
			DROP
			5 ROLL SIZE
			DATA_1 0x20 EQUALVERIFY
			7 PICK HASH256
			7 PICK HASH256 CAT
			HASH256 1 PICK CAT HASH256
			9 ROLL
			WHILE
				10 ROLL
				ROT CAT HASH256
				SWAP 1SUB
			ENDWHILE
			4 ROLL EQUALVERIFY
			DATA_2 0x5275
			5 PICK CATPUSHDATA
			7 PICK CATPUSHDATA
			SWAP CATPUSHDATA
			5 ROLL CATPUSHDATA
			1 CATPUSHDATA
			OUTPUTSCRIPT
			DATA_1 0x27 RIGHT
			CAT
			AMOUNT ASSET ROT
			RESERVEOUTPUT VERIFY
			2DROP 2DROP
			EVAL
		ENDIF
	DUP 5 EQUAL IF
		DROP
		8 PICK 7 PICK
		9 ROLL
		WHILE
			SWAP 10 ROLL SWAP
			CAT HASH256
			SWAP 1SUB
		ENDWHILE
		4 PICK EQUALVERIFY
		7 PICK NOTIF
			9 PICK
			11 PICK
			5 PICK NOTIF
				6 ROLL EQUALVERIFY
				3 ROLL EQUALVERIFY
				DROP
			ELSE
				HASH256 SWAP
				HASH256 CAT HASH256
				EQUALVERIFY
				NIP ROT DROP
			ENDIF
		ELSE
			DROP NIP ROT DROP
		ENDIF
		5 ROLL
		1SUB DUP 2MUL 6 ADD ROLL
		OVER 2MUL 7 ADD ROLL
		7 ROLL
		3 ROLL
		WHILE
			8 ROLL HASH256
			9 ROLL HASH256
			SWAP CAT HASH256
			ROT CAT HASH256
			SWAP 1SUB
		ENDWHILE
		DATA_2 0x5275
		6 PICK CATPUSHDATA
		ROT CATPUSHDATA
		SWAP CATPUSHDATA
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
	rightsHoldingContractHash [hash256.Size]byte
)

func init() {
	var err error
	rightsHoldingContract, err = txscript.ParseScriptString(rightsHoldingContractString)
	if err != nil {
		panic("failed parsing voting rights holding script: " + err.Error())
	}
	rightsHoldingContractHash = hash256.Sum(rightsHoldingContract)

	if pinnedRightsContractHash != bc.Hash(rightsHoldingContractHash).String() {
		panic(fmt.Sprintf("Expected right contract hash %s, current contract has hash %x",
			pinnedRightsContractHash, rightsHoldingContractHash[:]))
	}
}

// calculateOwnershipChain extends the provided chain of ownership with the provided
// holder and deadline using the formula:
//
//     Hash256(Hash256(Hash256(holder) + Hash256(deadline)) + oldchain)
//
func calculateOwnershipChain(oldChain bc.Hash, holder []byte, deadline int64) bc.Hash {
	h1 := hash256.Sum(holder)
	h2 := hash256.Sum(txscript.Int64ToScriptBytes(deadline))
	hash := hash256.Sum(append(h1[:], h2[:]...))
	data := append(hash[:], oldChain[:]...)
	return hash256.Sum(data)
}
