package vmutil

import (
	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

func IsUnspendable(prog []byte) bool {
	return len(prog) > 0 && prog[0] == byte(vm.OP_FAIL)
}

// TxMultiSigScript returns a valid script for a multisignature
// redemption where nrequired of the keys in pubkeys are required to
// have signed the transaction for success.  An ErrBadValue will
// be returned if nrequired is larger than the number of keys
// provided.
// The result is: <nrequired> <pubkey>... <npubkeys> 1 TXSIGHASH CHECKMULTISIG
func TxMultiSigScript(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	return doMultiSigScript(pubkeys, nrequired, false)
}

// BlockMultiSigScript is like TxMultiSigScript but for blocks.
// The result is: <nrequired> <pubkey>... <npubkeys> BLOCKSIGHASH CHECKMULTISIG
func BlockMultiSigScript(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	return doMultiSigScript(pubkeys, nrequired, true)
}

func doMultiSigScript(pubkeys []ed25519.PublicKey, nrequired int, isBlock bool) ([]byte, error) {
	if nrequired < 0 || len(pubkeys) < nrequired || (len(pubkeys) > 0 && nrequired == 0) {
		return nil, ErrBadValue
	}
	builder := NewBuilder()
	builder.AddInt64(int64(nrequired))
	for _, key := range pubkeys {
		builder.AddData(hd25519.PubBytes(key))
	}
	builder.AddInt64(int64(len(pubkeys)))
	if isBlock {
		builder.AddOp(vm.OP_BLOCKSIGHASH)
	} else {
		builder.AddInt64(1).AddOp(vm.OP_TXSIGHASH)
	}
	builder.AddOp(vm.OP_CHECKMULTISIG)
	return builder.Program, nil
}

// ParseTxMultiSigScript is the inverse of TxMultiSigScript().  It parses
// the script to produce the list of PublicKeys and nrequired values
// encoded within.
func ParseTxMultiSigScript(script []byte) ([]ed25519.PublicKey, int, error) {
	return doParseMultiSigScript(script, false)
}

func ParseBlockMultiSigScript(script []byte) ([]ed25519.PublicKey, int, error) {
	return doParseMultiSigScript(script, true)
}

func doParseMultiSigScript(script []byte, isBlock bool) ([]ed25519.PublicKey, int, error) {
	pops, err := vm.ParseProgram(script)
	if err != nil {
		return nil, 0, err
	}

	var minLen int
	if isBlock {
		minLen = 4
	} else {
		minLen = 5
	}

	if len(pops) < minLen {
		return nil, 0, vm.ErrShortProgram
	}

	if pops[len(pops)-1].Op != vm.OP_CHECKMULTISIG {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "no OP_CHECKMULTISIG")
	}

	nrequired, err := vm.AsInt64(pops[0].Data)
	if err != nil {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "parsing nrequired")
	}

	var npubkeysOpIndex int
	if isBlock {
		npubkeysOpIndex = len(pops) - 3
	} else {
		npubkeysOpIndex = len(pops) - 4
	}
	npubkeys, err := vm.AsInt64(pops[npubkeysOpIndex].Data)
	if err != nil {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "parsing npubkeys")
	}
	if npubkeys != int64(len(pops)-minLen) {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "npubkeys has wrong value")
	}
	if nrequired > npubkeys {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "nrequired > npubkeys")
	}
	if nrequired == 0 && npubkeys > 0 {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "nrequired == 0 and npubkeys > 0")
	}
	pubkeyPops := pops[1:npubkeysOpIndex]
	if !isPushOnly(pubkeyPops) {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "not push-only")
	}
	pubkeys := make([]ed25519.PublicKey, 0, len(pubkeyPops))
	for _, pop := range pubkeyPops {
		pubkey, err := hd25519.PubFromBytes(pop.Data)
		if err != nil {
			return nil, 0, errors.Wrap(ErrMultisigFormat, "could not parse pubkey")
		}
		pubkeys = append(pubkeys, pubkey)
	}
	return pubkeys, int(nrequired), nil
}

func isPushOnly(instructions []vm.Instruction) bool {
	for _, inst := range instructions {
		if len(inst.Data) > 0 {
			continue
		}
		if inst.Op == vm.OP_0 {
			continue
		}
		return false
	}
	return true
}

// SigsRequired returns the number of signatures required by
// script. Result is 1 unless script parses as a multisig script, in
// which case it's the number of sigs required by that.
func SigsRequired(script []byte) int {
	_, nsigs, err := ParseTxMultiSigScript(script)
	if err == nil {
		return nsigs
	}
	return 1
}

// PayToContractHash builds a contracthash-style p2c pkscript.
func PayToContractHash(contractHash bc.ContractHash, params [][]byte) []byte {
	builder := NewBuilder()
	for i := len(params) - 1; i >= 0; i-- {
		builder.AddData(params[i])
	}
	if len(params) > 0 {
		builder.AddInt64(int64(len(params))).AddOp(vm.OP_ROLL)
	}
	builder.AddOp(vm.OP_DUP).AddOp(vm.OP_SHA3).AddData(contractHash[:])
	builder.AddOp(vm.OP_EQUALVERIFY).AddOp(vm.OP_0).AddOp(vm.OP_CHECKPREDICATE)
	return builder.Program
}

// RedeemP2C builds program args for redeeming a contract.
func RedeemP2C(contract []byte, inputs [][]byte) [][]byte {
	args := make([][]byte, 0, len(inputs)+1)
	args = append(args, inputs...)
	if contract != nil {
		args = append(args, contract)
	}
	return args
}

// RedeemToPkScript takes a redeem script
// and calculates its corresponding pk script
func RedeemToPkScript(redeem []byte) []byte {
	hash := sha3.Sum256(redeem)
	builder := NewBuilder()
	builder.AddOp(vm.OP_DUP).AddOp(vm.OP_SHA3).AddData(hash[:]).AddOp(vm.OP_EQUALVERIFY)
	builder.AddOp(vm.OP_0).AddOp(vm.OP_CHECKPREDICATE)
	return builder.Program
}

func TxScripts(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, []byte, error) {
	redeem, err := TxMultiSigScript(pubkeys, nrequired)
	if err != nil {
		return nil, nil, err
	}
	return RedeemToPkScript(redeem), redeem, nil
}
