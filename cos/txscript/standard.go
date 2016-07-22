// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"

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

func IsMultiSig(script []byte) bool {
	pops, err := parseScript(script)
	if err != nil {
		return false
	}
	return isMultiSig(pops)
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

// SigsRequired returns the number of signatures required by
// script. Result is 1 unless script parses as a multisig script, in
// which case it's the number of sigs required by that.
func SigsRequired(script []byte) int {
	_, nsigs, err := ParseMultiSigScript(script)
	if err == nil {
		return nsigs
	}
	return 1
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
