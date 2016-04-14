// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"

	"chain/cos/bc"
	"chain/errors"
)

const (
	// MaxDataCarrierSize is the maximum number of bytes allowed in pushed
	// data to be considered a nulldata transaction
	MaxDataCarrierSize = 80

	// StandardVerifyFlags are the script flags which are used when
	// executing transaction scripts to enforce additional checks which
	// are required for the script to be considered standard.  These checks
	// help reduce issues related to transaction malleability as well as
	// allow pay-to-script hash transactions.  Note these flags are
	// different than what is required for the consensus rules in that they
	// are more strict.
	//
	// TODO: This definition does not belong here.  It belongs in a policy
	// package.
	StandardVerifyFlags = ScriptVerifyDERSignatures |
		ScriptVerifyStrictEncoding |
		ScriptVerifyMinimalData |
		ScriptStrictMultiSig |
		ScriptDiscourageUpgradableNops
)

// ScriptClass is an enumeration for the list of standard types of script.
type ScriptClass byte

// Classes of script payment known about in the blockchain.
const (
	NonStandardTy ScriptClass = iota // None of the recognized forms.
	PubKeyTy                         // Pay pubkey.
	PubKeyHashTy                     // Pay pubkey hash.
	ContractTy                       // Pay to contract.
	ScriptHashTy                     // Pay to script hash.
	MultiSigTy                       // Multi signature.
	NullDataTy                       // Empty data-only (provably prunable).
)

// scriptClassToName houses the human-readable strings which describe each
// script class.
var scriptClassToName = []string{
	NonStandardTy: "nonstandard",
	PubKeyTy:      "pubkey",
	PubKeyHashTy:  "pubkeyhash",
	ContractTy:    "contract",
	ScriptHashTy:  "scripthash",
	MultiSigTy:    "multisig",
	NullDataTy:    "nulldata",
}

// String implements the Stringer interface by returning the name of
// the enum script class. If the enum is invalid then "Invalid" will be
// returned.
func (t ScriptClass) String() string {
	if int(t) > len(scriptClassToName) || int(t) < 0 {
		return "Invalid"
	}
	return scriptClassToName[t]
}

// isPubkey returns true if the script passed is a pay-to-pubkey transaction,
// false otherwise.
func isPubkey(pops []parsedOpcode) bool {
	// Valid pubkeys are either 33 or 65 bytes.
	return len(pops) == 2 &&
		(len(pops[0].data) == 33 || len(pops[0].data) == 65) &&
		pops[1].opcode.value == OP_CHECKSIG
}

// isPubkeyHash returns true if the script passed is a pay-to-pubkey-hash
// transaction, false otherwise.
func isPubkeyHash(pops []parsedOpcode) bool {
	return len(pops) == 5 &&
		pops[0].opcode.value == OP_DUP &&
		pops[1].opcode.value == OP_HASH160 &&
		pops[2].opcode.value == OP_DATA_20 &&
		pops[3].opcode.value == OP_EQUALVERIFY &&
		pops[4].opcode.value == OP_CHECKSIG

}

// isMultiSig returns true if the passed script is a multisig transaction, false
// otherwise.
func isMultiSig(pops []parsedOpcode) bool {
	// The absolute minimum is 1 pubkey:
	// OP_0/OP_1-16 <pubkey> OP_1 OP_CHECKMULTISIG
	l := len(pops)
	if l < 4 {
		return false
	}
	if !isSmallInt(pops[0].opcode) {
		return false
	}
	if !isSmallInt(pops[l-2].opcode) {
		return false
	}
	if pops[l-1].opcode.value != OP_CHECKMULTISIG {
		return false
	}
	for _, pop := range pops[1 : l-2] {
		// Valid pubkeys are either 33 or 65 bytes.
		if len(pop.data) != 33 && len(pop.data) != 65 {
			return false
		}
	}
	return true
}

// isNullData returns true if the passed script is a null data transaction,
// false otherwise.
func isNullData(pops []parsedOpcode) bool {
	// A nulldata transaction is either a single OP_RETURN or an
	// OP_RETURN SMALLDATA (where SMALLDATA is a data push up to
	// MaxDataCarrierSize bytes).
	l := len(pops)
	if l == 1 && pops[0].opcode.value == OP_RETURN {
		return true
	}

	return l == 2 &&
		pops[0].opcode.value == OP_RETURN &&
		pops[1].opcode.value <= OP_PUSHDATA4 &&
		len(pops[1].data) <= MaxDataCarrierSize
}

// scriptType returns the type of the script being inspected from the known
// standard types.
func typeOfScript(pops []parsedOpcode) ScriptClass {
	if isPubkey(pops) {
		return PubKeyTy
	} else if isPubkeyHash(pops) {
		return PubKeyHashTy
	} else if isContract(pops) {
		return ContractTy
	} else if isScriptHash(pops) {
		return ScriptHashTy
	} else if isMultiSig(pops) {
		return MultiSigTy
	} else if isNullData(pops) {
		return NullDataTy
	}
	return NonStandardTy
}

// GetScriptClass returns the class of the script passed.
//
// NonStandardTy will be returned when the script does not parse.
func GetScriptClass(script []byte) ScriptClass {
	pops, err := parseScript(script)
	if err != nil {
		return NonStandardTy
	}
	return typeOfScript(pops)
}

// CalcMultiSigStats returns the number of public keys and signatures from
// a multi-signature transaction script.  The passed script MUST already be
// known to be a multi-signature script.
func CalcMultiSigStats(script []byte) (int, int, error) {
	pops, err := parseScript(script)
	if err != nil {
		return 0, 0, err
	}

	// A multi-signature script is of the pattern:
	//  NUM_SIGS PUBKEY PUBKEY PUBKEY... NUM_PUBKEYS OP_CHECKMULTISIG
	// Therefore the number of signatures is the oldest item on the stack
	// and the number of pubkeys is the 2nd to last.  Also, the absolute
	// minimum for a multi-signature script is 1 pubkey, so at least 4
	// items must be on the stack per:
	//  OP_1 PUBKEY OP_1 OP_CHECKMULTISIG
	if len(pops) < 4 {
		return 0, 0, ErrStackUnderflow
	}

	numSigs := asSmallInt(pops[0].opcode)
	numPubKeys := asSmallInt(pops[len(pops)-2].opcode)
	return numPubKeys, numSigs, nil
}

// payToPubKeyHashScript creates a new script to pay a transaction
// output to a 20-byte pubkey hash. It is expected that the input is a valid
// hash.
func payToPubKeyHashScript(pubKeyHash []byte) ([]byte, error) {
	return NewScriptBuilder().AddOp(OP_DUP).AddOp(OP_HASH160).
		AddData(pubKeyHash).AddOp(OP_EQUALVERIFY).AddOp(OP_CHECKSIG).
		Script()
}

// payToContractScript creates a new script to pay a transaction
// output to a contract.
func payToContractScript(contractHash, scriptVersion []byte, params [][]byte) ([]byte, error) {
	sb := NewScriptBuilder()
	sb = sb.AddData(scriptVersion).AddOp(OP_DROP)

	n := len(params)
	if n > 0 {
		for i := n - 1; i >= 0; i-- {
			sb = sb.AddData(params[i])
		}
		sb = sb.AddInt64(int64(n)).AddOp(OP_ROLL)
	}

	sb = sb.AddOp(OP_DUP).AddOp(OP_HASH256).AddData(contractHash).AddOp(OP_EQUALVERIFY).AddOp(OP_EVAL)
	return sb.Script()
}

// payToScriptHashScript creates a new script to pay a transaction output to a
// script hash. It is expected that the input is a valid hash.
func payToScriptHashScript(scriptHash []byte) ([]byte, error) {
	return NewScriptBuilder().AddOp(OP_HASH160).AddData(scriptHash).
		AddOp(OP_EQUAL).Script()
}

// payToPubkeyScript creates a new script to pay a transaction output to a
// public key. It is expected that the input is a valid pubkey.
func payToPubKeyScript(serializedPubKey []byte) ([]byte, error) {
	return NewScriptBuilder().AddData(serializedPubKey).
		AddOp(OP_CHECKSIG).Script()
}

// Satisfies btcutil.Address
type AddressContractHash struct {
	scriptVersion []byte
	params        [][]byte
	hash          bc.ContractHash
}

// String returns the string encoding of the transaction output
// destination.
//
// Please note that String differs subtly from EncodeAddress: String
// will return the value as a string without any conversion, while
// EncodeAddress may convert destination types (for example,
// converting pubkeys to P2PKH addresses) before encoding as a
// payment address string.
func (a *AddressContractHash) String() string {
	return a.EncodeAddress()
}

// EncodeAddress returns the string encoding of the payment address
// associated with the Address value.  See the comment on String
// for how this method differs from String.
func (a *AddressContractHash) EncodeAddress() string {
	return string(a.ScriptAddress())
}

// ScriptAddress returns the raw bytes of the address to be used
// when inserting the address into a txout's script.
func (a *AddressContractHash) ScriptAddress() []byte {
	result, _ := payToContractScript(a.hash[:], a.scriptVersion, a.params)
	return result
}

// IsForNet returns whether or not the address is associated with the
// passed bitcoin network.
func (a *AddressContractHash) IsForNet(*chaincfg.Params) bool {
	return true
}

func NewAddressContractHash(contractHash, scriptVersion []byte, params [][]byte) *AddressContractHash {
	result := AddressContractHash{scriptVersion: scriptVersion, params: params}
	copy(result.hash[:], contractHash)
	return &result
}

// PayToAddrScript creates a new script to pay a transaction output to
// the specified address.
func PayToAddrScript(addr btcutil.Address) ([]byte, error) {
	switch addr := addr.(type) {
	case *btcutil.AddressPubKeyHash:
		if addr == nil {
			return nil, ErrUnsupportedAddress
		}
		return payToPubKeyHashScript(addr.ScriptAddress())

	case *btcutil.AddressScriptHash:
		if addr == nil {
			return nil, ErrUnsupportedAddress
		}
		return payToScriptHashScript(addr.ScriptAddress())

	case *AddressContractHash:
		if addr == nil {
			return nil, ErrUnsupportedAddress
		}
		return addr.ScriptAddress(), nil

	case *btcutil.AddressPubKey:
		if addr == nil {
			return nil, ErrUnsupportedAddress
		}
		return payToPubKeyScript(addr.ScriptAddress())
	}

	return nil, ErrUnsupportedAddress
}

// MultiSigScript returns a valid script for a multisignature redemption where
// nrequired of the keys in pubkeys are required to have signed the transaction
// for success.  An ErrBadNumRequired will be returned if nrequired is larger
// than the number of keys provided.
func MultiSigScript(pubkeys []*btcutil.AddressPubKey, nrequired int) ([]byte, error) {
	if len(pubkeys) < nrequired {
		return nil, ErrBadNumRequired
	}

	builder := NewScriptBuilder().AddInt64(int64(nrequired))
	for _, key := range pubkeys {
		builder.AddData(key.ScriptAddress())
	}
	builder.AddInt64(int64(len(pubkeys)))
	builder.AddOp(OP_CHECKMULTISIG)

	return builder.Script()
}

// ParseMultiSigScript is (almost) the inverse of MultiSigScript().
// It parses the script to produce the list of PublicKeys and
// nrequired values encoded within.  (The "almost" is because
// MultiSigScript takes btcutil.AddressPubKeys, but this function
// gives back btcec.PublicKeys.)
func ParseMultiSigScript(script []byte) ([]*btcec.PublicKey, int, error) {
	pops, err := parseScript(script)
	if err != nil {
		return nil, 0, err
	}

	if len(pops) < 4 {
		return nil, 0, ErrStackShortScript // overloading this error code
	}
	nrequiredOp := pops[0].opcode
	if !isSmallInt(nrequiredOp) {
		return nil, 0, errors.Wrap(ErrScriptFormat, "nrequired not small int")
	}
	nrequired := asSmallInt(nrequiredOp)
	if nrequired < 1 {
		return nil, 0, errors.Wrap(ErrScriptFormat, "nrequired < 1")
	}
	if pops[len(pops)-1].opcode.value != OP_CHECKMULTISIG {
		return nil, 0, errors.Wrap(ErrScriptFormat, "no OP_CHECKMULTISIG")
	}
	npubkeysOp := pops[len(pops)-2].opcode
	if !isSmallInt(npubkeysOp) {
		return nil, 0, errors.Wrap(ErrScriptFormat, "npubkeys not small int")
	}
	npubkeys := asSmallInt(npubkeysOp)
	if npubkeys != len(pops)-3 {
		return nil, 0, errors.Wrap(ErrScriptFormat, "npubkeys has wrong value")
	}
	if nrequired > npubkeys {
		return nil, 0, errors.Wrap(ErrScriptFormat, "nrequired > npubkeys")
	}
	pubkeyPops := pops[1 : len(pops)-2]
	if !isPushOnly(pubkeyPops) {
		return nil, 0, errors.Wrap(ErrScriptFormat, "not push-only")
	}
	pubkeys := make([]*btcec.PublicKey, 0, len(pubkeyPops))
	for _, pop := range pubkeyPops {
		pubkeyData := pop.data
		pubkey, err := btcec.ParsePubKey(pubkeyData, btcec.S256())
		if err != nil {
			return nil, 0, errors.Wrap(err, "parsing pubkey")
		}
		pubkeys = append(pubkeys, pubkey)
	}
	return pubkeys, nrequired, nil
}

// PushedData returns an array of byte slices containing any pushed data found
// in the passed script.  This includes OP_0, OP_1 - OP_16, and OP_1NEGATE.
func PushedData(script []byte) ([][]byte, error) {
	pops, err := parseScript(script)
	if err != nil {
		return nil, err
	}

	var data [][]byte
	for _, pop := range pops {
		if isPushdataOp(pop) {
			data = append(data, asPushdata(pop))
		}
	}
	return data, nil
}
