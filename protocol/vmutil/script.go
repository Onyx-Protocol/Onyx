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

// BlockMultiSigScript returns a valid script for a multisignature
// consensus program where nrequired of the keys in pubkeys are
// required to have signed the block for success.  An ErrBadValue will
// be returned if nrequired is larger than the number of keys
// provided.
// The result is: BLOCKSIGHASH <pubkey>... <nrequired> <npubkeys> CHECKMULTISIG
func BlockMultiSigScript(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	if nrequired < 0 || len(pubkeys) < nrequired || (len(pubkeys) > 0 && nrequired == 0) {
		return nil, ErrBadValue
	}
	builder := NewBuilder()
	builder.AddOp(vm.OP_BLOCKSIGHASH)
	for _, key := range pubkeys {
		builder.AddData(hd25519.PubBytes(key))
	}
	builder.AddInt64(int64(nrequired)).AddInt64(int64(len(pubkeys))).AddOp(vm.OP_CHECKMULTISIG)
	return builder.Program, nil
}

func ParseBlockMultiSigScript(script []byte) ([]ed25519.PublicKey, int, error) {
	pops, err := vm.ParseProgram(script)
	if err != nil {
		return nil, 0, err
	}

	minLen := 4

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

	npubkeysOpIndex := len(pops) - 3

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

// RedeemToPkScript takes a redeem script
// and calculates its corresponding pk script
func RedeemToPkScript(redeem []byte) []byte {
	hash := sha3.Sum256(redeem)
	builder := NewBuilder()
	builder.AddOp(vm.OP_DUP).AddOp(vm.OP_SHA3).AddData(hash[:]).AddOp(vm.OP_EQUALVERIFY)
	builder.AddOp(vm.OP_0).AddOp(vm.OP_CHECKPREDICATE)
	return builder.Program
}

func P2DPMultiSigProgram(pubkeys []ed25519.PublicKey, nrequired int) []byte {
	builder := NewBuilder()
	// Expected stack: [... SIG SIG SIG PREDICATE]
	// Number of sigs must match nrequired.
	builder.AddOp(vm.OP_DUP).AddOp(vm.OP_TOALTSTACK) // stash a copy of the predicate
	builder.AddOp(vm.OP_SHA3)                        // stack is now [... SIG SIG SIG PREDICATEHASH]
	for _, p := range pubkeys {
		builder.AddData(hd25519.PubBytes(p))
	}
	builder.AddInt64(int64(nrequired))    // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M]
	builder.AddInt64(int64(len(pubkeys))) // stack is now [... sig sig sig PREDICATEHASH PUB PUB PUB M N]
	builder.AddOp(vm.OP_CHECKMULTISIG).AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK) // get the stashed predicate back
	builder.AddInt64(0).AddOp(vm.OP_CHECKPREDICATE)
	return builder.Program
}

func ParseP2DPMultiSigProgram(program []byte) ([]ed25519.PublicKey, int, error) {
	pops, err := vm.ParseProgram(program)
	if err != nil {
		return nil, 0, err
	}
	if len(pops) < 11 {
		return nil, 0, vm.ErrShortProgram
	}

	// Count all instructions backwards from the end in case there are
	// extra instructions at the beginning of the program (like a
	// <pushdata> DROP).

	npubkeys, err := vm.AsInt64(pops[len(pops)-6].Data)
	if err != nil {
		return nil, 0, err
	}
	if npubkeys <= 0 || int(npubkeys) > len(pops)-10 {
		return nil, 0, ErrBadValue
	}
	nrequired, err := vm.AsInt64(pops[len(pops)-7].Data)
	if err != nil {
		return nil, 0, err
	}
	if nrequired <= 0 {
		return nil, 0, ErrBadValue
	}

	firstPubkeyIndex := len(pops) - 7 - int(npubkeys)

	pubkeys := make([]ed25519.PublicKey, 0, npubkeys)
	for i := firstPubkeyIndex; i < firstPubkeyIndex+int(npubkeys); i++ {
		pubkey, err := hd25519.PubFromBytes(pops[i].Data)
		if err != nil {
			return nil, 0, err
		}
		pubkeys = append(pubkeys, pubkey)
	}
	return pubkeys, int(nrequired), nil
}
