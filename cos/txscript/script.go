// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chain/cos/bc"
	"chain/crypto/hash256"
)

// These are the constants specified for maximums in individual scripts.
const (
	MaxOpsPerScript       = 1000 // Max number of opcodes executed per script.
	MaxPubKeysPerMultiSig = 20   // Multisig can't have more sigs than this.
	MaxScriptElementSize  = 520  // Max bytes pushable to the stack.
)

var (
	ScriptVersion1 = []byte{0x1}
	ScriptVersion2 = []byte{0x2}
)

// isSmallInt returns whether or not the opcode is considered a small integer,
// which is an OP_0, or OP_1 through OP_16.
func isSmallInt(op *opcode) bool {
	if op.value == OP_0 || (op.value >= OP_1 && op.value <= OP_16) {
		return true
	}
	return false
}

// isScriptHash returns true if the script passed is a pay-to-script-hash
// transaction, false otherwise.
func isScriptHash(pops []parsedOpcode) bool {
	return len(pops) == 3 &&
		pops[0].opcode.value == OP_HASH160 &&
		pops[1].opcode.value == OP_DATA_20 &&
		pops[2].opcode.value == OP_EQUAL
}

// IsPayToScriptHash returns true if the script is in the standard
// pay-to-script-hash (P2SH) format, false otherwise.
func IsPayToScriptHash(script []byte) bool {
	pops, err := parseScript(script)
	if err != nil {
		return false
	}
	return isScriptHash(pops)
}

// PayToContractHash builds a contracthash-style p2c pkscript.
func PayToContractHash(contractHash bc.ContractHash, params []Item, scriptVersion []byte) ([]byte, error) {
	sb := payToContractHelper(params, scriptVersion)
	if len(params) > 0 {
		sb = sb.AddInt64(int64(len(params))).AddOp(OP_ROLL)
	}
	sb = sb.AddOp(OP_DUP).AddOp(OP_HASH256).AddData(contractHash[:])
	sb = sb.AddOp(OP_EQUALVERIFY).AddOp(OP_EVAL)
	return sb.Script()
}

// PayToContractInline builds an inline-style p2c pkscript.
func PayToContractInline(contract []byte, params []Item, scriptVersion []byte) ([]byte, error) {
	sb := payToContractHelper(params, scriptVersion)
	sb = sb.ConcatRawScript(contract)
	return sb.Script()
}

func payToContractHelper(params []Item, scriptVersion []byte) *ScriptBuilder {
	sb := NewScriptBuilder()
	sb = sb.AddData(scriptVersion).AddOp(OP_DROP)
	for i := len(params) - 1; i >= 0; i-- {
		sb = params[i].AddTo(sb)
	}
	return sb
}

var (
	ErrNotP2C      = errors.New("not in P2C format")
	ErrP2CMismatch = errors.New("contract mismatch")
)

// RedeemP2C builds a sigscript for redeeming the given contract.
func RedeemP2C(pkscript, contract []byte, inputs []Item) ([]byte, error) {
	scriptVersion, pkscriptContract, pkscriptContractHash, _ := ParseP2C(pkscript, contract)
	if scriptVersion == nil {
		return nil, ErrNotP2C
	}
	if pkscriptContract != nil {
		if !bytes.Equal(pkscriptContract, contract) {
			return nil, ErrP2CMismatch
		}
	} else {
		hash := hash256.Sum(contract)
		if hash != pkscriptContractHash {
			return nil, ErrP2CMismatch
		}
	}
	sb := NewScriptBuilder()
	for _, input := range inputs {
		sb = input.AddTo(sb)
	}
	if pkscriptContract == nil {
		sb = sb.AddData(contract)
	}
	return sb.Script()
}

// ParseP2C parses a p2c script.  It must have one of the following forms:
//   <scriptversion> DROP [<param N> <param N-1> ... <param 1>] contract-script...
//   <scriptversion> DROP [<param N> <param N-1> ... <param 1> <N> ROLL] DUP HASH256 <contract-hash> EQUALVERIFY EVAL
//
// Additionally, scriptversion must be a legal P2C version.
//
// If contractHint is non-nil, it's used as the expected contract to
// find at the end of an inline-style script, to help the parser
// distinguish the script from the params.  If contractHint is nil and
// the input is inline-style, then (HEURISTIC ALERT) the contract
// script must start with a non-pushdata op, again to distinguish the
// contract script from the params.
//
// If script has the first form, the return value is:
//   <scriptversion>, <the contract>, <a zero hash>, <the params>
// If script has the second form, the return value is:
//   <scriptversion>, nil, <the contract hash>, <the params>
// Otherwise the first return value is nil.
//
// TODO(bobg): Per kr's comment at
// https://github.com/chain-engineering/chain/pull/864#issuecomment-218553450,
// split up txscript into two packages: one for script execution
// semantics only, the other for script parsing and related functions.
func ParseP2C(script, contractHint []byte) (scriptVersion, contract []byte, contractHash bc.ContractHash, params [][]byte) {
	pops, err := parseScript(script)
	if err != nil {
		return nil, nil, contractHash, nil
	}
	return parseP2C(pops, script, contractHint)
}

var okVersions = [][]byte{ScriptVersion1, ScriptVersion2}

func parseP2C(pops []parsedOpcode, script, contractHint []byte) (scriptVersion, contract []byte, contractHash bc.ContractHash, params [][]byte) {
	scriptVersion = parseScriptVersion(pops)
	okVersion := false
	for _, v := range okVersions {
		if bytes.Equal(v, scriptVersion) {
			okVersion = true
			break
		}
	}
	if !okVersion {
		return nil, nil, contractHash, nil
	}

	isHashForm, contractHash, params := parseP2CHashForm(pops)
	if isHashForm {
		if contractHint != nil {
			expectedHash := hash256.Sum(contractHint)
			if expectedHash != contractHash {
				return nil, nil, contractHash, nil
			}
		}
		return scriptVersion, nil, contractHash, params
	}

	isInlineForm, contract, params := parseP2CInlineForm(pops, script, contractHint)
	if isInlineForm {
		return scriptVersion, contract, contractHash, params
	}

	return nil, nil, contractHash, nil
}

// Helper function for parseP2C.  Assumes scriptversion is already checked.
func parseP2CHashForm(pops []parsedOpcode) (isHashForm bool, contractHash bc.ContractHash, params [][]byte) {
	l := len(pops)

	if l < 7 ||
		(l > 7 && l < 10) ||
		pops[l-1].opcode.value != OP_EVAL ||
		pops[l-2].opcode.value != OP_EQUALVERIFY ||
		!isPushdataOp(pops[l-3]) ||
		len(pops[l-3].data) != hash256.Size ||
		pops[l-4].opcode.value != OP_HASH256 ||
		pops[l-5].opcode.value != OP_DUP {
		return false, contractHash, nil
	}

	if l > 7 {
		n, err := asScriptNum(pops[l-7], false)
		if err != nil ||
			n != scriptNum(l-9) ||
			pops[l-6].opcode.value != OP_ROLL ||
			!isPushdataOp(pops[l-7]) {
			return false, contractHash, nil
		}
		params = make([][]byte, 0, l-9)
		for i := l - 8; i >= 2; i-- {
			if !isPushdataOp(pops[i]) {
				return false, contractHash, nil
			}
			params = append(params, asPushdata(pops[i]))
		}
	}

	copy(contractHash[:], pops[l-3].data)

	return true, contractHash, params
}

// Helper function for parseP2C.  Assumes scriptversion is already checked.
func parseP2CInlineForm(pops []parsedOpcode, script, contractHint []byte) (isInlineForm bool, contract []byte, params [][]byte) {
	var nParams int

	if contractHint != nil {
		contractStart := len(script) - len(contractHint)
		if !bytes.Equal(script[contractStart:], contractHint) {
			return false, nil, nil
		}
		var err error
		pops, err = parseScript(script[:contractStart])
		if err != nil {
			return false, nil, nil
		}
		if len(pops) < 2 {
			return false, nil, nil
		}
		if pops[len(pops)-1].opcode.value == OP_NOP {
			pops = pops[:len(pops)-1]
		}
		for i := 2; i < len(pops); i++ {
			if !isPushdataOp(pops[i]) {
				return false, nil, nil
			}
		}
		nParams = len(pops) - 2
		contract = contractHint
	} else {
		for i := 2; i < len(pops) && isPushdataOp(pops[i]); i++ {
			nParams++
		}
		if nParams == len(pops)-2 {
			// No non-pushdata ops found
			return false, nil, nil
		}
		for i := 2 + nParams; i < len(pops); i++ {
			unparsedOpcode, err := pops[i].bytes()
			if err != nil {
				return false, nil, nil
			}
			contract = append(contract, unparsedOpcode...)
		}
		if len(contract) > 0 && contract[0] == OP_NOP {
			contract = contract[1:]
		}
	}

	if nParams > 0 {
		params = make([][]byte, 0, nParams)
		for i := 1 + nParams; i >= 2; i-- {
			params = append(params, asPushdata(pops[i]))
		}
	}

	return true, contract, params
}

// isPushOnly returns true if the script only pushes data, false otherwise.
func isPushOnly(pops []parsedOpcode) bool {
	// NOTE: This function does NOT verify opcodes directly since it is
	// internal and is only called with parsed opcodes for scripts that did
	// not have any parse errors.  Thus, consensus is properly maintained.

	for _, pop := range pops {
		if !isPushdataOp(pop) {
			return false
		}
	}
	return true
}

func isPushdataOp(pop parsedOpcode) bool {
	// All opcodes up to OP_16 are data push instructions.
	// NOTE: This does consider OP_RESERVED to be a data push
	// instruction, but execution of OP_RESERVED will fail anyways
	// and matches the behavior required by consensus.
	return pop.opcode.value <= OP_16
}

// asPushdata returns the pushdata for a data-pushing operation.  For
// OP_0 and OP_1 through OP_16, which have no pop.data field
// populated, this function constructs a data value.
func asPushdata(pop parsedOpcode) []byte {
	if pop.opcode.value == OP_0 {
		return nil
	}
	if pop.opcode.value >= OP_1 && pop.opcode.value <= OP_16 {
		return scriptNum(1 + pop.opcode.value - OP_1).Bytes()
	}
	if pop.opcode.value == OP_1NEGATE {
		return scriptNum(-1).Bytes()
	}
	return pop.data
}

// IsPushOnlyScript returns whether or not the passed script only pushes data.
//
// False will be returned when the script does not parse.
func IsPushOnlyScript(script []byte) bool {
	pops, err := parseScript(script)
	if err != nil {
		return false
	}
	return isPushOnly(pops)
}

// parseScriptTemplate is the same as parseScript but allows the passing of the
// template list for testing purposes.  When there are parse errors, it returns
// the list of parsed opcodes up to the point of failure along with the error.
func parseScriptTemplate(script []byte, opcodes *[256]opcode) ([]parsedOpcode, error) {
	retScript := make([]parsedOpcode, 0, len(script))
	for i := 0; i < len(script); {
		instr := script[i]
		op := opcodes[instr]
		pop := parsedOpcode{opcode: &op}

		// Parse data out of instruction.
		switch {
		// No additional data.  Note that some of the opcodes, notably
		// OP_1NEGATE, OP_0, and OP_[1-16] represent the data
		// themselves.
		case op.length == 1:
			i++

		// Data pushes of specific lengths -- OP_DATA_[1-75].
		case op.length > 1:
			if len(script[i:]) < op.length {
				return retScript, ErrStackShortScript
			}

			// Slice out the data.
			pop.data = script[i+1 : i+op.length]
			i += op.length

		// Data pushes with parsed lengths -- OP_PUSHDATAP{1,2,4}.
		case op.length < 0:
			var l uint
			off := i + 1

			if len(script[off:]) < -op.length {
				return retScript, ErrStackShortScript
			}

			// Next -length bytes are little endian length of data.
			switch op.length {
			case -1:
				l = uint(script[off])
			case -2:
				l = ((uint(script[off+1]) << 8) |
					uint(script[off]))
			case -4:
				l = ((uint(script[off+3]) << 24) |
					(uint(script[off+2]) << 16) |
					(uint(script[off+1]) << 8) |
					uint(script[off]))
			default:
				return retScript,
					fmt.Errorf("invalid opcode length %d",
						op.length)
			}

			// Move offset to beginning of the data.
			off += -op.length

			// Disallow entries that do not fit script or were
			// sign extended.
			if int(l) > len(script[off:]) || int(l) < 0 {
				return retScript, ErrStackShortScript
			}

			pop.data = script[off : off+int(l)]
			i += 1 - op.length + int(l)
		}

		retScript = append(retScript, pop)
	}

	return retScript, nil
}

// parseScript preparses the script in bytes into a list of parsedOpcodes while
// applying a number of sanity checks.
func parseScript(script []byte) ([]parsedOpcode, error) {
	return parseScriptTemplate(script, &opcodeArray)
}

// DisasmString formats a disassembled script for one line printing.  When the
// script fails to parse, the returned string will contain the disassembled
// script up to the point the failure occurred along with the string '[error]'
// appended.  In addition, the reason the script failed to parse is returned
// if the caller wants more information about the failure.
func DisasmString(buf []byte) (string, error) {
	disbuf := ""
	opcodes, err := parseScript(buf)
	for _, pop := range opcodes {
		disbuf += pop.print(true) + " "
	}
	if disbuf != "" {
		disbuf = disbuf[:len(disbuf)-1]
	}
	if err != nil {
		disbuf += "[error]"
	}
	return disbuf, err
}

// asSmallInt returns the passed opcode, which must be true according to
// isSmallInt(), as an integer.
func asSmallInt(op *opcode) int {
	if op.value == OP_0 {
		return 0
	}

	return int(op.value - (OP_1 - 1))
}

func asScriptNum(pop parsedOpcode, requireMinimal bool) (scriptNum, error) {
	if isSmallInt(pop.opcode) {
		return scriptNum(asSmallInt(pop.opcode)), nil
	}
	return makeScriptNum(pop.data, requireMinimal)
}

// IsUnspendable returns whether the passed public key script is unspendable, or
// guaranteed to fail at execution.  This allows inputs to be pruned instantly
// when entering the UTXO set.
func IsUnspendable(pkScript []byte) bool {
	pops, err := parseScript(pkScript)
	if err != nil {
		return true
	}

	return len(pops) > 0 && pops[0].opcode.value == OP_RETURN
}

// parse hex string into a []byte.
func parseHex(tok string) ([]byte, error) {
	if !strings.HasPrefix(tok, "0x") {
		return nil, errors.New("not a hex number")
	}
	return hex.DecodeString(tok[2:])
}

// shortFormOps holds a map of opcode names to values for use in short form
// parsing.  It is declared here so it only needs to be created once.
var shortFormOps map[string]byte

// ParseScriptString parses a string as as used in the Bitcoin Core
// reference tests into the script it came from.
//
// The format used for these tests is pretty simple if ad-hoc:
//   - Opcodes other than the push opcodes and unknown are present as
//     either OP_NAME or just NAME
//   - Plain numbers are made into push operations
//   - Numbers beginning with 0x are inserted into the []byte as-is (so
//     0x14 is OP_DATA_20)
//   - Single quoted strings are pushed as data
//   - Anything else is an error
//
// IMPORTANT! Equivalent script strings supplied to this function
// should always produce identical bytes.  Otherwise you risk
// producing a script-hash mismatch that could make some tx output
// depending on that hash unspendable.
func ParseScriptString(script string) ([]byte, error) {
	// Only create the short form opcode map once.
	if shortFormOps == nil {
		ops := make(map[string]byte)
		for opcodeName, opcodeValue := range OpcodeByName {
			if strings.Contains(opcodeName, "OP_UNKNOWN") {
				continue
			}
			ops[opcodeName] = opcodeValue

			// The opcodes named OP_# can't have the OP_ prefix
			// stripped or they would conflict with the plain
			// numbers.  Also, since OP_FALSE and OP_TRUE are
			// aliases for the OP_0, and OP_1, respectively, they
			// have the same value, so detect those by name and
			// allow them.
			if (opcodeName == "OP_FALSE" || opcodeName == "OP_TRUE") ||
				(opcodeValue != OP_0 && (opcodeValue < OP_1 ||
					opcodeValue > OP_16)) {

				ops[strings.TrimPrefix(opcodeName, "OP_")] = opcodeValue
			}
		}
		shortFormOps = ops
	}

	tokens := strings.Fields(script)
	builder := NewScriptBuilder()

	for _, tok := range tokens {
		if len(tok) == 0 {
			continue
		}
		// if parses as a plain number
		if num, err := strconv.ParseInt(tok, 10, 64); err == nil {
			builder.AddInt64(num)
			continue
		} else if bts, err := parseHex(tok); err == nil {
			builder.ConcatRawScript(bts)
		} else if len(tok) >= 2 &&
			tok[0] == '\'' && tok[len(tok)-1] == '\'' {
			builder.AddFullData([]byte(tok[1 : len(tok)-1]))
		} else if opcode, ok := shortFormOps[tok]; ok {
			builder.AddOp(opcode)
		} else {
			return nil, fmt.Errorf("bad token \"%s\"", tok)
		}

	}
	return builder.Script()
}

// ParseScriptVersion parses the version identifier from the script.
// The version is specified at the beginning of the script as
// [version] OP_DROP ...
// where [version] is any push-data op (including OP_0 and OP_1..OP_16).
//
// If the beginning of the script does not match this pattern, it's
// treated as OP_0 OP_DROP ...
func ParseScriptVersion(script []byte) ([]byte, error) {
	pops, err := parseScript(script)
	if err != nil {
		return nil, err
	}
	return parseScriptVersion(pops), nil
}

func parseScriptVersion(pops []parsedOpcode) []byte {
	if len(pops) < 2 {
		return nil
	}
	if pops[1].opcode.value != OP_DROP {
		return nil
	}
	if !isPushdataOp(pops[0]) {
		return nil
	}

	pop0 := pops[0]
	op0 := pop0.opcode
	data := pop0.data

	if data == nil && isSmallInt(op0) {
		return scriptNum(asSmallInt(op0)).Bytes()
	}
	return data
}
