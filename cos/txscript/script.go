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
)

// These are the constants specified for maximums in individual scripts.
const (
	MaxOpsPerScript       = 1000 // Max number of opcodes executed per script.
	MaxPubKeysPerMultiSig = 20   // Multisig can't have more sigs than this.
	MaxScriptElementSize  = 520  // Max bytes pushable to the stack.
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

// Returns true if the parsed script is in p2c format, false
// otherwise.
func isContract(pops []parsedOpcode) bool {
	c, _ := testContract(pops)
	return c != nil
}

type Contract struct {
	Hash          bc.ContractHash
	ScriptVersion []byte
}

func (c Contract) Match(hash bc.ContractHash, version []byte) bool {
	return c.Hash == hash && bytes.Equal(c.ScriptVersion, version)
}

// Returns true, the contract, and the params if the parsed script
// is in p2c format; false, nil, and nil otherwise.
//
// P2C format for N>=1 params is:
//   1 DROP <paramN> <paramN-1> ... <param1> <N> ROLL DUP HASH256 <contractHash> EQUALVERIFY EVAL
// For N=0 params it's just:
//   1 DROP DUP HASH256 <contractHash> EQUALVERIFY EVAL
func testContract(pops []parsedOpcode) (*Contract, [][]byte) {
	scriptVersionBytes := parseScriptVersion(pops)

	l := len(pops)
	if l < 7 || (l > 7 && l < 10) {
		// Zero-param form has exactly 7 opcodes.
		// 1+ params has 10 or more opcodes.
		return nil, nil
	}

	if pops[l-1].opcode.value != OP_EVAL {
		return nil, nil
	}
	if pops[l-2].opcode.value != OP_EQUALVERIFY {
		return nil, nil
	}

	if !isPushdataOp(pops[l-3]) {
		return nil, nil
	}
	if len(pops[l-3].data) != 32 {
		return nil, nil
	}

	if pops[l-4].opcode.value != OP_HASH256 {
		return nil, nil
	}
	if pops[l-5].opcode.value != OP_DUP {
		return nil, nil
	}

	var params [][]byte

	if l > 7 {
		params = make([][]byte, 0, l-9)

		if pops[l-6].opcode.value != OP_ROLL {
			return nil, nil
		}
		if !isPushdataOp(pops[l-7]) {
			return nil, nil
		}
		n, err := asScriptNum(pops[l-7], false)
		if err != nil {
			return nil, nil // swallow errors
		}
		if n != scriptNum(l-9) {
			return nil, nil
		}
		for i := l - 8; i >= 2; i-- {
			if !isPushdataOp(pops[i]) {
				return nil, nil
			}
			params = append(params, asPushdata(pops[i]))
		}
	}

	var contractHash bc.ContractHash
	copy(contractHash[:], pops[l-3].data)

	return &Contract{
		Hash:          contractHash,
		ScriptVersion: scriptVersionBytes,
	}, params
}

// IsPayToContract returns true if the script is in the standard
// pay-to-contract (P2C) format, false otherwise.
func IsPayToContract(script []byte) bool {
	contract, _ := TestPayToContract(script)
	return contract != nil
}

// TestPayToContract returns a Contract struct and the params if
// the script is in p2c format; nil and nil otherwise.
func TestPayToContract(script []byte) (*Contract, [][]byte) {
	pops, err := parseScript(script)
	if err != nil {
		return nil, nil
	}
	return testContract(pops)
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

// unparseScript reversed the action of parseScript and returns the
// parsedOpcodes as a list of bytes
func unparseScript(pops []parsedOpcode) ([]byte, error) {
	script := make([]byte, 0, len(pops))
	for _, pop := range pops {
		b, err := pop.bytes()
		if err != nil {
			return nil, err
		}
		script = append(script, b...)
	}
	return script, nil
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

// removeOpcode will remove any opcode matching ``opcode'' from the opcode
// stream in pkscript
func removeOpcode(pkscript []parsedOpcode, opcode byte) []parsedOpcode {
	retScript := make([]parsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if pop.opcode.value != opcode {
			retScript = append(retScript, pop)
		}
	}
	return retScript
}

// canonicalPush returns true if the object is either not a push instruction
// or the push instruction contained wherein is matches the canonical form
// or using the smallest instruction to do the job. False otherwise.
func canonicalPush(pop parsedOpcode) bool {
	opcode := pop.opcode.value
	data := pop.data
	dataLen := len(pop.data)
	if opcode > OP_16 {
		return true
	}

	if opcode < OP_PUSHDATA1 && opcode > OP_0 && (dataLen == 1 && data[0] <= 16) {
		return false
	}
	if opcode == OP_PUSHDATA1 && dataLen < OP_PUSHDATA1 {
		return false
	}
	if opcode == OP_PUSHDATA2 && dataLen <= 0xff {
		return false
	}
	if opcode == OP_PUSHDATA4 && dataLen <= 0xffff {
		return false
	}
	return true
}

// removeOpcodeByData will return the script minus any opcodes that would push
// the passed data to the stack.
func removeOpcodeByData(pkscript []parsedOpcode, data []byte) []parsedOpcode {
	retScript := make([]parsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if !canonicalPush(pop) || !bytes.Contains(pop.data, data) {
			retScript = append(retScript, pop)
		}
	}
	return retScript

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
