package vm

import (
	"encoding/binary"
	"fmt"
	"math"

	"chain/errors"
	"chain/math/checked"
)

type Op uint8

func (op Op) String() string {
	return ops[op].name
}

type Instruction struct {
	Op   Op
	Len  uint32
	Data []byte
}

const (
	OP_FALSE Op = 0x00
	OP_0     Op = 0x00 // synonym

	OP_1    Op = 0x51
	OP_TRUE Op = 0x51 // synonym

	OP_2  Op = 0x52
	OP_3  Op = 0x53
	OP_4  Op = 0x54
	OP_5  Op = 0x55
	OP_6  Op = 0x56
	OP_7  Op = 0x57
	OP_8  Op = 0x58
	OP_9  Op = 0x59
	OP_10 Op = 0x5a
	OP_11 Op = 0x5b
	OP_12 Op = 0x5c
	OP_13 Op = 0x5d
	OP_14 Op = 0x5e
	OP_15 Op = 0x5f
	OP_16 Op = 0x60

	OP_DATA_1  Op = 0x01
	OP_DATA_2  Op = 0x02
	OP_DATA_3  Op = 0x03
	OP_DATA_4  Op = 0x04
	OP_DATA_5  Op = 0x05
	OP_DATA_6  Op = 0x06
	OP_DATA_7  Op = 0x07
	OP_DATA_8  Op = 0x08
	OP_DATA_9  Op = 0x09
	OP_DATA_10 Op = 0x0a
	OP_DATA_11 Op = 0x0b
	OP_DATA_12 Op = 0x0c
	OP_DATA_13 Op = 0x0d
	OP_DATA_14 Op = 0x0e
	OP_DATA_15 Op = 0x0f
	OP_DATA_16 Op = 0x10
	OP_DATA_17 Op = 0x11
	OP_DATA_18 Op = 0x12
	OP_DATA_19 Op = 0x13
	OP_DATA_20 Op = 0x14
	OP_DATA_21 Op = 0x15
	OP_DATA_22 Op = 0x16
	OP_DATA_23 Op = 0x17
	OP_DATA_24 Op = 0x18
	OP_DATA_25 Op = 0x19
	OP_DATA_26 Op = 0x1a
	OP_DATA_27 Op = 0x1b
	OP_DATA_28 Op = 0x1c
	OP_DATA_29 Op = 0x1d
	OP_DATA_30 Op = 0x1e
	OP_DATA_31 Op = 0x1f
	OP_DATA_32 Op = 0x20
	OP_DATA_33 Op = 0x21
	OP_DATA_34 Op = 0x22
	OP_DATA_35 Op = 0x23
	OP_DATA_36 Op = 0x24
	OP_DATA_37 Op = 0x25
	OP_DATA_38 Op = 0x26
	OP_DATA_39 Op = 0x27
	OP_DATA_40 Op = 0x28
	OP_DATA_41 Op = 0x29
	OP_DATA_42 Op = 0x2a
	OP_DATA_43 Op = 0x2b
	OP_DATA_44 Op = 0x2c
	OP_DATA_45 Op = 0x2d
	OP_DATA_46 Op = 0x2e
	OP_DATA_47 Op = 0x2f
	OP_DATA_48 Op = 0x30
	OP_DATA_49 Op = 0x31
	OP_DATA_50 Op = 0x32
	OP_DATA_51 Op = 0x33
	OP_DATA_52 Op = 0x34
	OP_DATA_53 Op = 0x35
	OP_DATA_54 Op = 0x36
	OP_DATA_55 Op = 0x37
	OP_DATA_56 Op = 0x38
	OP_DATA_57 Op = 0x39
	OP_DATA_58 Op = 0x3a
	OP_DATA_59 Op = 0x3b
	OP_DATA_60 Op = 0x3c
	OP_DATA_61 Op = 0x3d
	OP_DATA_62 Op = 0x3e
	OP_DATA_63 Op = 0x3f
	OP_DATA_64 Op = 0x40
	OP_DATA_65 Op = 0x41
	OP_DATA_66 Op = 0x42
	OP_DATA_67 Op = 0x43
	OP_DATA_68 Op = 0x44
	OP_DATA_69 Op = 0x45
	OP_DATA_70 Op = 0x46
	OP_DATA_71 Op = 0x47
	OP_DATA_72 Op = 0x48
	OP_DATA_73 Op = 0x49
	OP_DATA_74 Op = 0x4a
	OP_DATA_75 Op = 0x4b

	OP_PUSHDATA1 Op = 0x4c
	OP_PUSHDATA2 Op = 0x4d
	OP_PUSHDATA4 Op = 0x4e
	OP_1NEGATE   Op = 0x4f
	OP_NOP       Op = 0x61

	OP_JUMP           Op = 0x63
	OP_JUMPIF         Op = 0x64
	OP_VERIFY         Op = 0x69
	OP_FAIL           Op = 0x6a
	OP_CHECKPREDICATE Op = 0xc0

	OP_TOALTSTACK   Op = 0x6b
	OP_FROMALTSTACK Op = 0x6c
	OP_2DROP        Op = 0x6d
	OP_2DUP         Op = 0x6e
	OP_3DUP         Op = 0x6f
	OP_2OVER        Op = 0x70
	OP_2ROT         Op = 0x71
	OP_2SWAP        Op = 0x72
	OP_IFDUP        Op = 0x73
	OP_DEPTH        Op = 0x74
	OP_DROP         Op = 0x75
	OP_DUP          Op = 0x76
	OP_NIP          Op = 0x77
	OP_OVER         Op = 0x78
	OP_PICK         Op = 0x79
	OP_ROLL         Op = 0x7a
	OP_ROT          Op = 0x7b
	OP_SWAP         Op = 0x7c
	OP_TUCK         Op = 0x7d

	OP_CAT         Op = 0x7e
	OP_SUBSTR      Op = 0x7f
	OP_LEFT        Op = 0x80
	OP_RIGHT       Op = 0x81
	OP_SIZE        Op = 0x82
	OP_CATPUSHDATA Op = 0x89

	OP_INVERT      Op = 0x83
	OP_AND         Op = 0x84
	OP_OR          Op = 0x85
	OP_XOR         Op = 0x86
	OP_EQUAL       Op = 0x87
	OP_EQUALVERIFY Op = 0x88

	OP_1ADD               Op = 0x8b
	OP_1SUB               Op = 0x8c
	OP_2MUL               Op = 0x8d
	OP_2DIV               Op = 0x8e
	OP_NEGATE             Op = 0x8f
	OP_ABS                Op = 0x90
	OP_NOT                Op = 0x91
	OP_0NOTEQUAL          Op = 0x92
	OP_ADD                Op = 0x93
	OP_SUB                Op = 0x94
	OP_MUL                Op = 0x95
	OP_DIV                Op = 0x96
	OP_MOD                Op = 0x97
	OP_LSHIFT             Op = 0x98
	OP_RSHIFT             Op = 0x99
	OP_BOOLAND            Op = 0x9a
	OP_BOOLOR             Op = 0x9b
	OP_NUMEQUAL           Op = 0x9c
	OP_NUMEQUALVERIFY     Op = 0x9d
	OP_NUMNOTEQUAL        Op = 0x9e
	OP_LESSTHAN           Op = 0x9f
	OP_GREATERTHAN        Op = 0xa0
	OP_LESSTHANOREQUAL    Op = 0xa1
	OP_GREATERTHANOREQUAL Op = 0xa2
	OP_MIN                Op = 0xa3
	OP_MAX                Op = 0xa4
	OP_WITHIN             Op = 0xa5

	OP_SHA256        Op = 0xa8
	OP_SHA3          Op = 0xaa
	OP_CHECKSIG      Op = 0xac
	OP_CHECKMULTISIG Op = 0xad
	OP_TXSIGHASH     Op = 0xae
	OP_BLOCKHASH     Op = 0xaf

	OP_CHECKOUTPUT Op = 0xc1
	OP_ASSET       Op = 0xc2
	OP_AMOUNT      Op = 0xc3
	OP_PROGRAM     Op = 0xc4
	OP_MINTIME     Op = 0xc5
	OP_MAXTIME     Op = 0xc6
	OP_TXDATA      Op = 0xc7
	OP_ENTRYDATA   Op = 0xc8
	OP_INDEX       Op = 0xc9
	OP_ENTRYID     Op = 0xca
	OP_OUTPUTID    Op = 0xcb
	OP_NONCE       Op = 0xcc
	OP_NEXTPROGRAM Op = 0xcd
	OP_BLOCKTIME   Op = 0xce
)

type opInfo struct {
	op   Op
	name string
	fn   func(*virtualMachine) error
}

var (
	ops = [256]opInfo{
		// data pushing
		OP_FALSE: {OP_FALSE, "FALSE", opFalse},

		// sic: the PUSHDATA ops all share an implementation
		OP_PUSHDATA1: {OP_PUSHDATA1, "PUSHDATA1", opPushdata},
		OP_PUSHDATA2: {OP_PUSHDATA2, "PUSHDATA2", opPushdata},
		OP_PUSHDATA4: {OP_PUSHDATA4, "PUSHDATA4", opPushdata},

		OP_1NEGATE: {OP_1NEGATE, "1NEGATE", op1Negate},

		OP_NOP: {OP_NOP, "NOP", opNop},

		// control flow
		OP_JUMP:   {OP_JUMP, "JUMP", opJump},
		OP_JUMPIF: {OP_JUMPIF, "JUMPIF", opJumpIf},

		OP_VERIFY: {OP_VERIFY, "VERIFY", opVerify},
		OP_FAIL:   {OP_FAIL, "FAIL", opFail},

		OP_TOALTSTACK:   {OP_TOALTSTACK, "TOALTSTACK", opToAltStack},
		OP_FROMALTSTACK: {OP_FROMALTSTACK, "FROMALTSTACK", opFromAltStack},
		OP_2DROP:        {OP_2DROP, "2DROP", op2Drop},
		OP_2DUP:         {OP_2DUP, "2DUP", op2Dup},
		OP_3DUP:         {OP_3DUP, "3DUP", op3Dup},
		OP_2OVER:        {OP_2OVER, "2OVER", op2Over},
		OP_2ROT:         {OP_2ROT, "2ROT", op2Rot},
		OP_2SWAP:        {OP_2SWAP, "2SWAP", op2Swap},
		OP_IFDUP:        {OP_IFDUP, "IFDUP", opIfDup},
		OP_DEPTH:        {OP_DEPTH, "DEPTH", opDepth},
		OP_DROP:         {OP_DROP, "DROP", opDrop},
		OP_DUP:          {OP_DUP, "DUP", opDup},
		OP_NIP:          {OP_NIP, "NIP", opNip},
		OP_OVER:         {OP_OVER, "OVER", opOver},
		OP_PICK:         {OP_PICK, "PICK", opPick},
		OP_ROLL:         {OP_ROLL, "ROLL", opRoll},
		OP_ROT:          {OP_ROT, "ROT", opRot},
		OP_SWAP:         {OP_SWAP, "SWAP", opSwap},
		OP_TUCK:         {OP_TUCK, "TUCK", opTuck},

		OP_CAT:         {OP_CAT, "CAT", opCat},
		OP_SUBSTR:      {OP_SUBSTR, "SUBSTR", opSubstr},
		OP_LEFT:        {OP_LEFT, "LEFT", opLeft},
		OP_RIGHT:       {OP_RIGHT, "RIGHT", opRight},
		OP_SIZE:        {OP_SIZE, "SIZE", opSize},
		OP_CATPUSHDATA: {OP_CATPUSHDATA, "CATPUSHDATA", opCatpushdata},

		OP_INVERT:      {OP_INVERT, "INVERT", opInvert},
		OP_AND:         {OP_AND, "AND", opAnd},
		OP_OR:          {OP_OR, "OR", opOr},
		OP_XOR:         {OP_XOR, "XOR", opXor},
		OP_EQUAL:       {OP_EQUAL, "EQUAL", opEqual},
		OP_EQUALVERIFY: {OP_EQUALVERIFY, "EQUALVERIFY", opEqualVerify},

		OP_1ADD:               {OP_1ADD, "1ADD", op1Add},
		OP_1SUB:               {OP_1SUB, "1SUB", op1Sub},
		OP_2MUL:               {OP_2MUL, "2MUL", op2Mul},
		OP_2DIV:               {OP_2DIV, "2DIV", op2Div},
		OP_NEGATE:             {OP_NEGATE, "NEGATE", opNegate},
		OP_ABS:                {OP_ABS, "ABS", opAbs},
		OP_NOT:                {OP_NOT, "NOT", opNot},
		OP_0NOTEQUAL:          {OP_0NOTEQUAL, "0NOTEQUAL", op0NotEqual},
		OP_ADD:                {OP_ADD, "ADD", opAdd},
		OP_SUB:                {OP_SUB, "SUB", opSub},
		OP_MUL:                {OP_MUL, "MUL", opMul},
		OP_DIV:                {OP_DIV, "DIV", opDiv},
		OP_MOD:                {OP_MOD, "MOD", opMod},
		OP_LSHIFT:             {OP_LSHIFT, "LSHIFT", opLshift},
		OP_RSHIFT:             {OP_RSHIFT, "RSHIFT", opRshift},
		OP_BOOLAND:            {OP_BOOLAND, "BOOLAND", opBoolAnd},
		OP_BOOLOR:             {OP_BOOLOR, "BOOLOR", opBoolOr},
		OP_NUMEQUAL:           {OP_NUMEQUAL, "NUMEQUAL", opNumEqual},
		OP_NUMEQUALVERIFY:     {OP_NUMEQUALVERIFY, "NUMEQUALVERIFY", opNumEqualVerify},
		OP_NUMNOTEQUAL:        {OP_NUMNOTEQUAL, "NUMNOTEQUAL", opNumNotEqual},
		OP_LESSTHAN:           {OP_LESSTHAN, "LESSTHAN", opLessThan},
		OP_GREATERTHAN:        {OP_GREATERTHAN, "GREATERTHAN", opGreaterThan},
		OP_LESSTHANOREQUAL:    {OP_LESSTHANOREQUAL, "LESSTHANOREQUAL", opLessThanOrEqual},
		OP_GREATERTHANOREQUAL: {OP_GREATERTHANOREQUAL, "GREATERTHANOREQUAL", opGreaterThanOrEqual},
		OP_MIN:                {OP_MIN, "MIN", opMin},
		OP_MAX:                {OP_MAX, "MAX", opMax},
		OP_WITHIN:             {OP_WITHIN, "WITHIN", opWithin},

		OP_SHA256:        {OP_SHA256, "SHA256", opSha256},
		OP_SHA3:          {OP_SHA3, "SHA3", opSha3},
		OP_CHECKSIG:      {OP_CHECKSIG, "CHECKSIG", opCheckSig},
		OP_CHECKMULTISIG: {OP_CHECKMULTISIG, "CHECKMULTISIG", opCheckMultiSig},
		OP_TXSIGHASH:     {OP_TXSIGHASH, "TXSIGHASH", opTxSigHash},
		OP_BLOCKHASH:     {OP_BLOCKHASH, "BLOCKHASH", opBlockHash},

		OP_CHECKOUTPUT: {OP_CHECKOUTPUT, "CHECKOUTPUT", opCheckOutput},
		OP_ASSET:       {OP_ASSET, "ASSET", opAsset},
		OP_AMOUNT:      {OP_AMOUNT, "AMOUNT", opAmount},
		OP_PROGRAM:     {OP_PROGRAM, "PROGRAM", opProgram},
		OP_MINTIME:     {OP_MINTIME, "MINTIME", opMinTime},
		OP_MAXTIME:     {OP_MAXTIME, "MAXTIME", opMaxTime},
		OP_TXDATA:      {OP_TXDATA, "TXDATA", opTxData},
		OP_ENTRYDATA:   {OP_ENTRYDATA, "ENTRYDATA", opEntryData},
		OP_INDEX:       {OP_INDEX, "INDEX", opIndex},
		OP_ENTRYID:     {OP_ENTRYID, "ENTRYID", opEntryID},
		OP_OUTPUTID:    {OP_OUTPUTID, "OUTPUTID", opOutputID},
		OP_NONCE:       {OP_NONCE, "NONCE", opNonce},
		OP_NEXTPROGRAM: {OP_NEXTPROGRAM, "NEXTPROGRAM", opNextProgram},
		OP_BLOCKTIME:   {OP_BLOCKTIME, "BLOCKTIME", opBlockTime},
	}

	opsByName map[string]opInfo
)

// ParseOp parses the op at position pc in prog, returning the parsed
// instruction (opcode plus any associated data).
func ParseOp(prog []byte, pc uint32) (inst Instruction, err error) {
	if len(prog) > math.MaxInt32 {
		err = ErrLongProgram
	}
	l := uint32(len(prog))
	if pc >= l {
		err = ErrShortProgram
		return
	}
	opcode := Op(prog[pc])
	inst.Op = opcode
	inst.Len = 1
	if opcode >= OP_1 && opcode <= OP_16 {
		inst.Data = []byte{uint8(opcode-OP_1) + 1}
		return
	}
	if opcode >= OP_DATA_1 && opcode <= OP_DATA_75 {
		inst.Len += uint32(opcode - OP_DATA_1 + 1)
		end, ok := checked.AddUint32(pc, inst.Len)
		if !ok {
			err = errors.WithDetail(checked.ErrOverflow, "data length exceeds max program size")
			return
		}
		if end > l {
			err = ErrShortProgram
			return
		}
		inst.Data = prog[pc+1 : end]
		return
	}
	if opcode == OP_PUSHDATA1 {
		if pc == l-1 {
			err = ErrShortProgram
			return
		}
		n := prog[pc+1]
		inst.Len += uint32(n) + 1
		end, ok := checked.AddUint32(pc, inst.Len)
		if !ok {
			err = errors.WithDetail(checked.ErrOverflow, "data length exceeds max program size")
		}
		if end > l {
			err = ErrShortProgram
			return
		}
		inst.Data = prog[pc+2 : end]
		return
	}
	if opcode == OP_PUSHDATA2 {
		if len(prog) < 3 || pc > l-3 {
			err = ErrShortProgram
			return
		}
		n := binary.LittleEndian.Uint16(prog[pc+1 : pc+3])
		inst.Len += uint32(n) + 2
		end, ok := checked.AddUint32(pc, inst.Len)
		if !ok {
			err = errors.WithDetail(checked.ErrOverflow, "data length exceeds max program size")
			return
		}
		if end > l {
			err = ErrShortProgram
			return
		}
		inst.Data = prog[pc+3 : end]
		return
	}
	if opcode == OP_PUSHDATA4 {
		if len(prog) < 5 || pc > l-5 {
			err = ErrShortProgram
			return
		}
		inst.Len += 4

		n := binary.LittleEndian.Uint32(prog[pc+1 : pc+5])
		var ok bool
		inst.Len, ok = checked.AddUint32(inst.Len, n)
		if !ok {
			err = errors.WithDetail(checked.ErrOverflow, "data length exceeds max program size")
			return
		}
		end, ok := checked.AddUint32(pc, inst.Len)
		if !ok {
			err = errors.WithDetail(checked.ErrOverflow, "data length exceeds max program size")
			return
		}
		if end > l {
			err = ErrShortProgram
			return
		}
		inst.Data = prog[pc+5 : end]
		return
	}
	if opcode == OP_JUMP || opcode == OP_JUMPIF {
		inst.Len += 4
		end, ok := checked.AddUint32(pc, inst.Len)
		if !ok {
			err = errors.WithDetail(checked.ErrOverflow, "jump target exceeds max program size")
			return
		}
		if end > l {
			err = ErrShortProgram
			return
		}
		inst.Data = prog[pc+1 : end]
		return
	}
	return
}

func ParseProgram(prog []byte) ([]Instruction, error) {
	var result []Instruction
	for pc := uint32(0); pc < uint32(len(prog)); { // update pc inside the loop
		inst, err := ParseOp(prog, pc)
		if err != nil {
			return nil, err
		}
		result = append(result, inst)
		var ok bool
		pc, ok = checked.AddUint32(pc, inst.Len)
		if !ok {
			return nil, errors.WithDetail(checked.ErrOverflow, "program counter exceeds max program size")
		}
	}
	return result, nil
}

var isExpansion [256]bool

func init() {
	for i := 1; i <= 75; i++ {
		ops[i] = opInfo{Op(i), fmt.Sprintf("DATA_%d", i), opPushdata}
	}
	for i := uint8(0); i <= 15; i++ {
		op := uint8(OP_1) + i
		ops[op] = opInfo{Op(op), fmt.Sprintf("%d", i+1), opPushdata}
	}

	// This is here to break a dependency cycle
	ops[OP_CHECKPREDICATE] = opInfo{OP_CHECKPREDICATE, "CHECKPREDICATE", opCheckPredicate}

	opsByName = make(map[string]opInfo)
	for _, info := range ops {
		opsByName[info.name] = info
	}
	opsByName["0"] = ops[OP_FALSE]
	opsByName["TRUE"] = ops[OP_1]

	for i := 0; i <= 255; i++ {
		if ops[i].name == "" {
			ops[i] = opInfo{Op(i), fmt.Sprintf("NOPx%02x", i), opNop}
			isExpansion[i] = true
		}
	}
}
