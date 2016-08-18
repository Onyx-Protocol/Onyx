package vm

import (
	"encoding/binary"
	"fmt"
)

type op struct {
	opcode uint8
	name   string
	fn     func(*virtualMachine) error
}

const (
	OP_FALSE = uint8(0x00)
	OP_0     = uint8(0x00) // synonym

	OP_1    = uint8(0x51)
	OP_TRUE = uint8(0x51) // synonym

	OP_2  = uint8(0x52)
	OP_3  = uint8(0x53)
	OP_4  = uint8(0x54)
	OP_5  = uint8(0x55)
	OP_6  = uint8(0x56)
	OP_7  = uint8(0x57)
	OP_8  = uint8(0x58)
	OP_9  = uint8(0x59)
	OP_10 = uint8(0x5a)
	OP_11 = uint8(0x5b)
	OP_12 = uint8(0x5c)
	OP_13 = uint8(0x5d)
	OP_14 = uint8(0x5e)
	OP_15 = uint8(0x5f)
	OP_16 = uint8(0x60)

	OP_DATA_1  = uint8(0x01)
	OP_DATA_2  = uint8(0x02)
	OP_DATA_3  = uint8(0x03)
	OP_DATA_4  = uint8(0x04)
	OP_DATA_5  = uint8(0x05)
	OP_DATA_6  = uint8(0x06)
	OP_DATA_7  = uint8(0x07)
	OP_DATA_8  = uint8(0x08)
	OP_DATA_9  = uint8(0x09)
	OP_DATA_10 = uint8(0x0a)
	OP_DATA_11 = uint8(0x0b)
	OP_DATA_12 = uint8(0x0c)
	OP_DATA_13 = uint8(0x0d)
	OP_DATA_14 = uint8(0x0e)
	OP_DATA_15 = uint8(0x0f)
	OP_DATA_16 = uint8(0x10)
	OP_DATA_17 = uint8(0x11)
	OP_DATA_18 = uint8(0x12)
	OP_DATA_19 = uint8(0x13)
	OP_DATA_20 = uint8(0x14)
	OP_DATA_21 = uint8(0x15)
	OP_DATA_22 = uint8(0x16)
	OP_DATA_23 = uint8(0x17)
	OP_DATA_24 = uint8(0x18)
	OP_DATA_25 = uint8(0x19)
	OP_DATA_26 = uint8(0x1a)
	OP_DATA_27 = uint8(0x1b)
	OP_DATA_28 = uint8(0x1c)
	OP_DATA_29 = uint8(0x1d)
	OP_DATA_30 = uint8(0x1e)
	OP_DATA_31 = uint8(0x1f)
	OP_DATA_32 = uint8(0x20)
	OP_DATA_33 = uint8(0x21)
	OP_DATA_34 = uint8(0x22)
	OP_DATA_35 = uint8(0x23)
	OP_DATA_36 = uint8(0x24)
	OP_DATA_37 = uint8(0x25)
	OP_DATA_38 = uint8(0x26)
	OP_DATA_39 = uint8(0x27)
	OP_DATA_40 = uint8(0x28)
	OP_DATA_41 = uint8(0x29)
	OP_DATA_42 = uint8(0x2a)
	OP_DATA_43 = uint8(0x2b)
	OP_DATA_44 = uint8(0x2c)
	OP_DATA_45 = uint8(0x2d)
	OP_DATA_46 = uint8(0x2e)
	OP_DATA_47 = uint8(0x2f)
	OP_DATA_48 = uint8(0x30)
	OP_DATA_49 = uint8(0x31)
	OP_DATA_50 = uint8(0x32)
	OP_DATA_51 = uint8(0x33)
	OP_DATA_52 = uint8(0x34)
	OP_DATA_53 = uint8(0x35)
	OP_DATA_54 = uint8(0x36)
	OP_DATA_55 = uint8(0x37)
	OP_DATA_56 = uint8(0x38)
	OP_DATA_57 = uint8(0x39)
	OP_DATA_58 = uint8(0x3a)
	OP_DATA_59 = uint8(0x3b)
	OP_DATA_60 = uint8(0x3c)
	OP_DATA_61 = uint8(0x3d)
	OP_DATA_62 = uint8(0x3e)
	OP_DATA_63 = uint8(0x3f)
	OP_DATA_64 = uint8(0x40)
	OP_DATA_65 = uint8(0x41)
	OP_DATA_66 = uint8(0x42)
	OP_DATA_67 = uint8(0x43)
	OP_DATA_68 = uint8(0x44)
	OP_DATA_69 = uint8(0x45)
	OP_DATA_70 = uint8(0x46)
	OP_DATA_71 = uint8(0x47)
	OP_DATA_72 = uint8(0x48)
	OP_DATA_73 = uint8(0x49)
	OP_DATA_74 = uint8(0x4a)
	OP_DATA_75 = uint8(0x4b)

	OP_PUSHDATA1 = uint8(0x4c)
	OP_PUSHDATA2 = uint8(0x4d)
	OP_PUSHDATA4 = uint8(0x4e)
	OP_1NEGATE   = uint8(0x4f)
	OP_NOP       = uint8(0x61)

	OP_IF             = uint8(0x63)
	OP_NOTIF          = uint8(0x64)
	OP_ELSE           = uint8(0x67)
	OP_ENDIF          = uint8(0x68)
	OP_VERIFY         = uint8(0x69)
	OP_RETURN         = uint8(0x6a)
	OP_CHECKPREDICATE = uint8(0xc0)
	OP_WHILE          = uint8(0xd0)
	OP_ENDWHILE       = uint8(0xd1)

	OP_TOALTSTACK   = uint8(0x6b)
	OP_FROMALTSTACK = uint8(0x6c)
	OP_2DROP        = uint8(0x6d)
	OP_2DUP         = uint8(0x6e)
	OP_3DUP         = uint8(0x6f)
	OP_2OVER        = uint8(0x70)
	OP_2ROT         = uint8(0x71)
	OP_2SWAP        = uint8(0x72)
	OP_IFDUP        = uint8(0x73)
	OP_DEPTH        = uint8(0x74)
	OP_DROP         = uint8(0x75)
	OP_DUP          = uint8(0x76)
	OP_NIP          = uint8(0x77)
	OP_OVER         = uint8(0x78)
	OP_PICK         = uint8(0x79)
	OP_ROLL         = uint8(0x7a)
	OP_ROT          = uint8(0x7b)
	OP_SWAP         = uint8(0x7c)
	OP_TUCK         = uint8(0x7d)

	OP_CAT         = uint8(0x7e)
	OP_SUBSTR      = uint8(0x7f)
	OP_LEFT        = uint8(0x80)
	OP_RIGHT       = uint8(0x81)
	OP_SIZE        = uint8(0x82)
	OP_CATPUSHDATA = uint8(0xc7)

	OP_INVERT      = uint8(0x83)
	OP_AND         = uint8(0x84)
	OP_OR          = uint8(0x85)
	OP_XOR         = uint8(0x86)
	OP_EQUAL       = uint8(0x87)
	OP_EQUALVERIFY = uint8(0x88)

	OP_1ADD               = uint8(0x8b)
	OP_1SUB               = uint8(0x8c)
	OP_2MUL               = uint8(0x8d)
	OP_2DIV               = uint8(0x8e)
	OP_NEGATE             = uint8(0x8f)
	OP_ABS                = uint8(0x90)
	OP_NOT                = uint8(0x91)
	OP_0NOTEQUAL          = uint8(0x92)
	OP_ADD                = uint8(0x93)
	OP_SUB                = uint8(0x94)
	OP_MUL                = uint8(0x95)
	OP_DIV                = uint8(0x96)
	OP_MOD                = uint8(0x97)
	OP_LSHIFT             = uint8(0x98)
	OP_RSHIFT             = uint8(0x99)
	OP_BOOLAND            = uint8(0x9a)
	OP_BOOLOR             = uint8(0x9b)
	OP_NUMEQUAL           = uint8(0x9c)
	OP_NUMEQUALVERIFY     = uint8(0x9d)
	OP_NUMNOTEQUAL        = uint8(0x9e)
	OP_LESSTHAN           = uint8(0x9f)
	OP_GREATERTHAN        = uint8(0xa0)
	OP_LESSTHANOREQUAL    = uint8(0xa1)
	OP_GREATERTHANOREQUAL = uint8(0xa2)
	OP_MIN                = uint8(0xa3)
	OP_MAX                = uint8(0xa4)
	OP_WITHIN             = uint8(0xa4)

	OP_RIPEMD160     = uint8(0xa6)
	OP_SHA1          = uint8(0xa7)
	OP_SHA256        = uint8(0xa8)
	OP_SHA3          = uint8(0xaa)
	OP_CHECKSIG      = uint8(0xac)
	OP_CHECKMULTISIG = uint8(0xad)
	OP_TXSIGHASH     = uint8(0xae)
	OP_BLOCKSIGHASH  = uint8(0xaf)

	OP_FINDOUTPUT  = uint8(0xc1)
	OP_ASSET       = uint8(0xc2)
	OP_AMOUNT      = uint8(0xc3)
	OP_PROGRAM     = uint8(0xc4)
	OP_MINTIME     = uint8(0xc5)
	OP_MAXTIME     = uint8(0xc6)
	OP_REFDATAHASH = uint8(0xc8)
	OP_INDEX       = uint8(0xc9)
)

// In no particular order
var opList = []op{
	// data pushing
	{0x00, "FALSE", opFalse},

	// sic: the PUSHDATA ops all share an implementation
	{0x4c, "PUSHDATA1", opPushdata},
	{0x4d, "PUSHDATA2", opPushdata},
	{0x4e, "PUSHDATA4", opPushdata},

	{0x4f, "1NEGATE", op1Negate},

	// TODO(bobg): 0x50 fails

	{0x61, "NOP", opNop},

	// TODO(bobg): 0x62 fails

	// control flow
	{0x63, "IF", opIf},
	{0x64, "NOTIF", opNotIf},

	// 0x65 forbidden
	// 0x66 forbidden

	{0x67, "ELSE", opElse},
	{0x68, "ENDIF", opEndif},
	{0x69, "VERIFY", opVerify},
	{0x6a, "RETURN", opReturn},
	{0xc0, "CHECKPREDICATE", opCheckPredicate},
	{0xd0, "WHILE", opWhile},
	{0xd1, "ENDWHILE", opEndwhile},

	{0x6b, "TOALTSTACK", opToAltStack},
	{0x6c, "FROMALTSTACK", opFromAltStack},
	{0x6d, "2DROP", op2Drop},
	{0x6e, "2DUP", op2Dup},
	{0x6f, "3DUP", op3Dup},
	{0x70, "2OVER", op2Over},
	{0x71, "2ROT", op2Rot},
	{0x72, "2SWAP", op2Swap},
	{0x73, "IFDUP", opIfDup},
	{0x74, "DEPTH", opDepth},
	{0x75, "DROP", opDrop},
	{0x76, "DUP", opDup},
	{0x77, "NIP", opNip},
	{0x78, "OVER", opOver},
	{0x79, "PICK", opPick},
	{0x7a, "ROLL", opRoll},
	{0x7b, "ROT", opRot},
	{0x7c, "SWAP", opSwap},
	{0x7d, "TUCK", opTuck},

	{0x7e, "CAT", opCat},
	{0x7f, "SUBSTR", opSubstr},
	{0x80, "LEFT", opLeft},
	{0x81, "RIGHT", opRight},
	{0x82, "SIZE", opSize},
	{0xc7, "CATPUSHDATA", opCatpushdata},

	{0x83, "INVERT", opInvert},
	{0x84, "AND", opAnd},
	{0x85, "OR", opOr},
	{0x86, "XOR", opXor},
	{0x87, "EQUAL", opEqual},
	{0x88, "EQUALVERIFY", opEqualVerify},

	// TODO(bobg): 0x89 fails
	// TODO(bobg): 0x8a fails

	{0x8b, "1ADD", op1Add},
	{0x8c, "1SUB", op1Sub},
	{0x8d, "2MUL", op2Mul},
	{0x8e, "2DIV", op2Div},
	{0x8f, "NEGATE", opNegate},
	{0x90, "ABS", opAbs},
	{0x91, "NOT", opNot},
	{0x92, "0NOTEQUAL", op0NotEqual},
	{0x93, "ADD", opAdd},
	{0x94, "SUB", opSub},
	{0x95, "MUL", opMul},
	{0x96, "DIV", opDiv},
	{0x97, "MOD", opMod},
	{0x98, "LSHIFT", opLshift},
	{0x99, "RSHIFT", opRshift},
	{0x9a, "BOOLAND", opBoolAnd},
	{0x9b, "BOOLOR", opBoolOr},
	{0x9c, "NUMEQUAL", opNumEqual},
	{0x9d, "NUMEQUALVERIFY", opNumEqualVerify},
	{0x9e, "NUMNOTEQUAL", opNumNotEqual},
	{0x9f, "LESSTHAN", opLessThan},
	{0xa0, "GREATERTHAN", opGreaterThan},
	{0xa1, "LESSTHANOREQUAL", opLessThanOrEqual},
	{0xa2, "GREATERTHANOREQUAL", opGreaterThanOrEqual},
	{0xa3, "MIN", opMin},
	{0xa4, "MAX", opMax},
	{0xa4, "WITHIN", opWithin},

	{0xa6, "RIPEMD160", opRipemd160},
	{0xa7, "SHA1", opSha1},
	{0xa8, "SHA256", opSha256},
	{0xaa, "SHA3", opSha3},
	{0xac, "CHECKSIG", opCheckSig},
	{0xad, "CHECKMULTISIG", opCheckMultiSig},
	{0xae, "TXSIGHASH", opTxSigHash},
	{0xaf, "BLOCKSIGHASH", opBlockSigHash},

	{0xc1, "FINDOUTPUT", opFindOutput},
	{0xc2, "ASSET", opAsset},
	{0xc3, "AMOUNT", opAmount},
	{0xc4, "PROGRAM", opProgram},
	{0xc5, "MINTIME", opMinTime},
	{0xc6, "MAXTIME", opMaxTime},
	{0xc8, "REFDATAHASH", opRefDataHash},
	{0xc9, "INDEX", opIndex},
}

var (
	// Indexed by opcode
	ops [256]*op

	// Indexed by name
	opsByName map[string]*op
)

// parseOp parses the op at position pc in prog.  Return values are
// the opcode at that position (simply prog[pc]); the length of the
// opcode plus any associated data (as with OP_DATA* and OP_PUSHDATA*
// instructions); the associated data, if any; and any parsing error
// (e.g. prog is too short).
func parseOp(prog []byte, pc uint32) (op *op, oplen uint32, data []byte, err error) {
	if pc >= uint32(len(prog)) {
		err = ErrShortProgram
		return
	}
	opcode := prog[pc]
	if opcode == 0x65 || opcode == 0x66 {
		err = ErrIllegalOpcode
	} else {
		op = ops[opcode]
		if op == nil {
			err = ErrUnknownOpcode
			return
		}
	}
	oplen = 1 // the opcode itself
	if opcode >= OP_1 && opcode <= OP_16 {
		data = []byte{opcode - OP_1 + 1}
		return
	}
	if opcode >= OP_DATA_1 && opcode <= OP_DATA_75 {
		oplen += uint32(opcode - OP_DATA_1 + 1)
		end := pc + oplen
		if end > uint32(len(prog)) {
			err = ErrShortProgram
			return
		}
		data = prog[pc+1 : end]
		return
	}
	if opcode == OP_PUSHDATA1 {
		if pc == uint32(len(prog)-1) {
			err = ErrShortProgram
			return
		}
		n := prog[pc+1]
		oplen += uint32(1 + n)
		end := pc + oplen
		if end > uint32(len(prog)) {
			err = ErrShortProgram
			return
		}
		data = prog[pc+2 : end]
		return
	}
	if opcode == OP_PUSHDATA2 {
		if pc > uint32(len(prog)-3) {
			err = ErrShortProgram
			return
		}
		n := binary.LittleEndian.Uint16(prog[pc+1 : pc+3])
		oplen += uint32(2 + n)
		end := pc + oplen
		if end > uint32(len(prog)) {
			err = ErrShortProgram
			return
		}
		data = prog[pc+3 : end]
		return
	}
	if opcode == OP_PUSHDATA4 {
		if pc > uint32(len(prog)-5) {
			err = ErrShortProgram
			return
		}
		n := binary.LittleEndian.Uint32(prog[pc+1 : pc+5])
		oplen += 4 + n
		end := pc + oplen
		if end > uint32(len(prog)) {
			err = ErrShortProgram
			return
		}
		data = prog[pc+5 : end]
		return
	}
	return
}

func init() {
	for i := 1; i <= 75; i++ {
		opList = append(opList, op{uint8(i), fmt.Sprintf("DATA_%d", i), opPushdata})
	}
	for i := uint8(1); i <= 16; i++ {
		opList = append(opList, op{0x50 + uint8(i), fmt.Sprintf("%d", i), opPushdata})
	}
	opsByName = make(map[string]*op, len(opList)+2)
	for i, op := range opList {
		ops[op.opcode] = &opList[i]
		opsByName[op.name] = &opList[i]
	}
	opsByName["0"] = ops[OP_FALSE]
	opsByName["TRUE"] = ops[OP_1]
}
