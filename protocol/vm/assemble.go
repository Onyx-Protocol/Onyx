package vm

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"chain/errors"
)

// Assemble converts a string like "2 3 ADD 5 NUMEQUAL" into 0x525393559c.
// The input should not include PUSHDATA (or OP_<num>) ops; those will
// be inferred.
// Input may include jump-target labels of the form $foo, which can
// then be used as JUMP:$foo or JUMPIF:$foo.
func Assemble(s string) (res []byte, err error) {
	// maps labels to the location each refers to
	locations := make(map[string]uint32)

	// maps unresolved uses of labels to the locations that need to be filled in
	unresolved := make(map[string][]int)

	handleJump := func(addrStr string, opcode Op) error {
		res = append(res, byte(opcode))
		l := len(res)

		var fourBytes [4]byte
		res = append(res, fourBytes[:]...)

		if strings.HasPrefix(addrStr, "$") {
			unresolved[addrStr] = append(unresolved[addrStr], l)
			return nil
		}

		address, err := strconv.ParseUint(addrStr, 10, 32)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(res[l:], uint32(address))
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Split(split)
	for scanner.Scan() {
		token := scanner.Text()
		if info, ok := opsByName[token]; ok {
			if strings.HasPrefix(token, "PUSHDATA") || strings.HasPrefix(token, "JUMP") {
				return nil, errors.Wrap(ErrToken, token)
			}
			res = append(res, byte(info.op))
		} else if strings.HasPrefix(token, "JUMP:") {
			// TODO (Dan): add IF/ELSE/ENDIF and BEGIN/WHILE/REPEAT
			err = handleJump(strings.TrimPrefix(token, "JUMP:"), OP_JUMP)
			if err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(token, "JUMPIF:") {
			err = handleJump(strings.TrimPrefix(token, "JUMPIF:"), OP_JUMPIF)
			if err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(token, "$") {
			if _, seen := locations[token]; seen {
				return nil, fmt.Errorf("label %s redefined", token)
			}
			if len(res) > math.MaxInt32 {
				return nil, fmt.Errorf("program too long")
			}
			locations[token] = uint32(len(res))
		} else if strings.HasPrefix(token, "0x") {
			bytes, err := hex.DecodeString(strings.TrimPrefix(token, "0x"))
			if err != nil {
				return nil, err
			}
			res = append(res, PushdataBytes(bytes)...)
		} else if len(token) >= 2 && token[0] == '\'' && token[len(token)-1] == '\'' {
			bytes := make([]byte, 0, len(token)-2)
			var b int
			for i := 1; i < len(token)-1; i++ {
				if token[i] == '\\' {
					i++
				}
				bytes = append(bytes, token[i])
				b++
			}
			res = append(res, PushdataBytes(bytes)...)
		} else if num, err := strconv.ParseInt(token, 10, 64); err == nil {
			res = append(res, PushdataInt64(num)...)
		} else {
			return nil, errors.Wrap(ErrToken, token)
		}
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	for label, uses := range unresolved {
		location, ok := locations[label]
		if !ok {
			return nil, fmt.Errorf("undefined label %s", label)
		}
		for _, use := range uses {
			binary.LittleEndian.PutUint32(res[use:], location)
		}
	}

	return res, nil
}

func Disassemble(prog []byte) (string, error) {
	var (
		insts []Instruction

		// maps program locations (used as jump targets) to a label for each
		labels = make(map[uint32]string)
	)

	// first pass: look for jumps
	for i := uint32(0); i < uint32(len(prog)); {
		inst, err := ParseOp(prog, i)
		if err != nil {
			return "", err
		}
		switch inst.Op {
		case OP_JUMP, OP_JUMPIF:
			addr := binary.LittleEndian.Uint32(inst.Data)
			if _, ok := labels[addr]; !ok {
				labelNum := len(labels)
				label := words[labelNum%len(words)]
				if labelNum >= len(words) {
					label += fmt.Sprintf("%d", labelNum/len(words)+1)
				}
				labels[addr] = label
			}
		}
		insts = append(insts, inst)
		i += inst.Len
	}

	var (
		loc  uint32
		strs []string
	)

	for _, inst := range insts {
		if label, ok := labels[loc]; ok {
			strs = append(strs, "$"+label)
		}

		var str string
		switch inst.Op {
		case OP_JUMP, OP_JUMPIF:
			addr := binary.LittleEndian.Uint32(inst.Data)
			str = fmt.Sprintf("%s:$%s", inst.Op.String(), labels[addr])
		default:
			if len(inst.Data) > 0 {
				str = fmt.Sprintf("0x%x", inst.Data)
			} else {
				str = inst.Op.String()
			}
		}
		strs = append(strs, str)

		loc += inst.Len
	}

	if label, ok := labels[loc]; ok {
		strs = append(strs, "$"+label)
	}

	return strings.Join(strs, " "), nil
}

// split is a bufio.SplitFunc for scanning the input to Compile.
// It starts like bufio.ScanWords but adjusts the return value to
// account for quoted strings.
func split(inp []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanWords(inp, atEOF)
	if err != nil {
		return
	}
	if len(token) > 1 && token[0] != '\'' {
		return
	}
	var start int
	for ; start < len(inp); start++ {
		if !unicode.IsSpace(rune(inp[start])) {
			break
		}
	}
	if start == len(inp) || inp[start] != '\'' {
		return
	}
	var escape bool
	for i := start + 1; i < len(inp); i++ {
		if escape {
			escape = false
		} else {
			switch inp[i] {
			case '\'':
				advance = i + 1
				token = inp[start:advance]
				return
			case '\\':
				escape = true
			}
		}
	}
	// Reached the end of the input with no closing quote.
	if atEOF {
		return 0, nil, ErrToken
	}
	return 0, nil, nil
}

var words = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
	"quebec", "romeo", "sierra", "tango", "uniform", "victor", "whisky", "xray",
	"yankee", "zulu",
}
