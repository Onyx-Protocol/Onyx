package vmutil

import (
	"chain/crypto/ed25519"
	"chain/errors"
	"chain/protocol/vm"
)

var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

func IsUnspendable(prog []byte) bool {
	return len(prog) > 0 && prog[0] == byte(vm.OP_FAIL)
}

// BlockMultiSigProgram returns a valid multisignature consensus
// program where nrequired of the keys in pubkeys are required to have
// signed the block for success.  An ErrBadValue will be returned if
// nrequired is larger than the number of keys provided.  The result
// is: BLOCKHASH <pubkey>... <nrequired> <npubkeys> CHECKMULTISIG
func BlockMultiSigProgram(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	err := checkMultiSigParams(int64(nrequired), int64(len(pubkeys)))
	if err != nil {
		return nil, err
	}
	builder := NewBuilder()
	builder.AddOp(vm.OP_BLOCKHASH)
	for _, key := range pubkeys {
		builder.AddData(key)
	}
	builder.AddInt64(int64(nrequired)).AddInt64(int64(len(pubkeys))).AddOp(vm.OP_CHECKMULTISIG)
	return builder.Build()
}

func ParseBlockMultiSigProgram(script []byte) ([]ed25519.PublicKey, int, error) {
	pops, err := vm.ParseProgram(script)
	if err != nil {
		return nil, 0, err
	}
	if len(pops) < 4 {
		return nil, 0, vm.ErrShortProgram
	}
	if pops[len(pops)-1].Op != vm.OP_CHECKMULTISIG {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "no OP_CHECKMULTISIG")
	}
	npubkeys, err := vm.AsInt64(pops[len(pops)-2].Data)
	if err != nil {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "parsing npubkeys")
	}
	if int(npubkeys) != len(pops)-4 {
		return nil, 0, vm.ErrShortProgram
	}
	nrequired, err := vm.AsInt64(pops[len(pops)-3].Data)
	if err != nil {
		return nil, 0, errors.Wrap(ErrMultisigFormat, "parsing nrequired")
	}
	err = checkMultiSigParams(nrequired, npubkeys)
	if err != nil {
		return nil, 0, err
	}

	firstPubkeyIndex := len(pops) - 3 - int(npubkeys)

	pubkeys := make([]ed25519.PublicKey, 0, npubkeys)
	for i := firstPubkeyIndex; i < firstPubkeyIndex+int(npubkeys); i++ {
		if len(pops[i].Data) != ed25519.PublicKeySize {
			return nil, 0, err
		}
		pubkeys = append(pubkeys, ed25519.PublicKey(pops[i].Data))
	}
	return pubkeys, int(nrequired), nil
}

func P2SPMultiSigProgram(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	err := checkMultiSigParams(int64(nrequired), int64(len(pubkeys)))
	if err != nil {
		return nil, err
	}
	builder := NewBuilder()
	// Expected stack: [... NARGS SIG SIG SIG PREDICATE]
	// Number of sigs must match nrequired.
	builder.AddOp(vm.OP_DUP).AddOp(vm.OP_TOALTSTACK) // stash a copy of the predicate
	builder.AddOp(vm.OP_SHA3)                        // stack is now [... NARGS SIG SIG SIG PREDICATEHASH]
	for _, p := range pubkeys {
		builder.AddData(p)
	}
	builder.AddInt64(int64(nrequired))                     // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M]
	builder.AddInt64(int64(len(pubkeys)))                  // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M N]
	builder.AddOp(vm.OP_CHECKMULTISIG).AddOp(vm.OP_VERIFY) // stack is now [... NARGS]
	builder.AddOp(vm.OP_FROMALTSTACK)                      // stack is now [... NARGS PREDICATE]
	builder.AddInt64(0).AddOp(vm.OP_CHECKPREDICATE)
	return builder.Build()
}

func ParseP2SPMultiSigProgram(program []byte) ([]ed25519.PublicKey, int, error) {
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
	if int(npubkeys) > len(pops)-10 {
		return nil, 0, vm.ErrShortProgram
	}
	nrequired, err := vm.AsInt64(pops[len(pops)-7].Data)
	if err != nil {
		return nil, 0, err
	}
	err = checkMultiSigParams(nrequired, npubkeys)
	if err != nil {
		return nil, 0, err
	}

	firstPubkeyIndex := len(pops) - 7 - int(npubkeys)

	pubkeys := make([]ed25519.PublicKey, 0, npubkeys)
	for i := firstPubkeyIndex; i < firstPubkeyIndex+int(npubkeys); i++ {
		if len(pops[i].Data) != ed25519.PublicKeySize {
			return nil, 0, err
		}
		pubkeys = append(pubkeys, ed25519.PublicKey(pops[i].Data))
	}
	return pubkeys, int(nrequired), nil
}

func checkMultiSigParams(nrequired, npubkeys int64) error {
	if nrequired < 0 {
		return errors.WithDetail(ErrBadValue, "negative quorum")
	}
	if npubkeys < 0 {
		return errors.WithDetail(ErrBadValue, "negative pubkey count")
	}
	if nrequired > npubkeys {
		return errors.WithDetail(ErrBadValue, "quorum too big")
	}
	if nrequired == 0 && npubkeys > 0 {
		return errors.WithDetail(ErrBadValue, "quorum empty with non-empty pubkey list")
	}
	return nil
}
