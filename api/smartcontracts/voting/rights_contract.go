package voting

import (
	"bytes"
	"crypto/sha256"

	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/hash256"
)

// scriptVersion encodes the version of the scripting language required
// for executing the voting rights contract.
var scriptVersion = []byte{0x02}

const (
	infiniteDeadline = 0x7fffffff
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
	var (
		params [][]byte
	)

	params = append(params, txscript.BoolToScriptBytes(r.Delegatable))
	params = append(params, txscript.Int64ToScriptBytes(r.Deadline))
	params = append(params, r.OwnershipChain[:])
	params = append(params, r.HolderScript)
	params = append(params, r.AdminScript)

	addr := txscript.NewAddressContractHash(rightsHoldingContractHash[:], scriptVersion, params)
	return addr.ScriptAddress()
}

// testRightsContract tests whether the given pkscript is a voting
// rights holding contract.
func testRightsContract(pkscript []byte) (*rightScriptData, error) {
	contract, params := txscript.TestPayToContract(pkscript)
	if contract == nil {
		return nil, nil
	}
	if !contract.Match(rightsHoldingContractHash, scriptVersion) {
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
// and the ownership hash to rewind to for recall clauses.
func testRightsSigscript(sigscript []byte) (ok bool, c rightsContractClause, ownershipHash bc.Hash) {
	data, err := txscript.PushedData(sigscript)
	if err != nil {
		return false, c, ownershipHash
	}
	if len(data) < 2 {
		return false, c, ownershipHash
	}
	if !bytes.Equal(data[len(data)-1], rightsHoldingContract) {
		return false, c, ownershipHash
	}

	clauseBytes := data[len(data)-2]
	if len(clauseBytes) != 1 {
		return false, c, ownershipHash
	}
	c = rightsContractClause(clauseBytes[0])
	if c < clauseAuthenticate || c > clauseCancel {
		return false, c, ownershipHash
	}

	// If it's not a recall, early exit.
	if c != clauseRecall {
		return true, c, ownershipHash
	}

	// Extract the ownership chain for the recall clause.
	if len(data) < 3 || len(data[len(data)-3]) != cap(ownershipHash) {
		return false, c, ownershipHash
	}
	copy(ownershipHash[:], data[len(data)-3])
	return true, c, ownershipHash
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
	// 5 - Override     (Unimplemented)
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
			TOALTSTACK
			EVAL VERIFY
			FROMALTSTACK
			EVAL
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
			TOALTSTACK
			EVAL VERIFY
			FROMALTSTACK
			EVAL
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
			ENDWHILE DROP
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
			2DROP DROP
			SWAP TOALTSTACK
			EVAL VERIFY
			FROMALTSTACK
			EVAL
		ENDIF
	`
)

var (
	rightsHoldingContract     []byte
	rightsHoldingContractHash [sha256.Size]byte
)

func init() {
	var err error
	rightsHoldingContract, err = txscript.ParseScriptString(rightsHoldingContractString)
	if err != nil {
		panic("failed parsing voting rights holding script: " + err.Error())
	}
	// TODO(jackson): Before going to production, we'll probably want to hard-code the
	// contract hash and panic if the contract changes.
	rightsHoldingContractHash = hash256.Sum(rightsHoldingContract)
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
