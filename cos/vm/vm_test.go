package vm

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

type tracebuf struct {
	bytes.Buffer
}

func (t tracebuf) dump() {
	os.Stdout.Write(t.Bytes())
}

// Programs that run without error and return a true result.
func TestProgramOK(t *testing.T) {
	doOKNotOK(t, true)
}

// Programs that run without error and return a false result.
func TestProgramNotOK(t *testing.T) {
	doOKNotOK(t, false)
}

func doOKNotOK(t *testing.T, expectOK bool) {
	cases := []struct {
		prog string
		args [][]byte
	}{
		{"TRUE", nil},

		// bitwise ops
		{"INVERT 0xfef0 EQUAL", [][]byte{{0x01, 0x0f}}},

		{"AND 0x02 EQUAL", [][]byte{{0x03}, {0x06}}},
		{"AND 0x02 EQUAL", [][]byte{{0x03, 0xff}, {0x06}}},

		{"OR 0x07 EQUAL", [][]byte{{0x03}, {0x06}}},
		{"OR 0x07ff EQUAL", [][]byte{{0x03, 0xff}, {0x06}}},

		{"XOR 0x05 EQUAL", [][]byte{{0x03}, {0x06}}},
		{"XOR 0x05ff EQUAL", [][]byte{{0x03, 0xff}, {0x06}}},

		// numeric and logical ops
		{"1ADD 2 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"1ADD 0 NUMEQUAL", [][]byte{Int64Bytes(-1)}},

		{"1SUB 1 NUMEQUAL", [][]byte{Int64Bytes(2)}},
		{"1SUB -1 NUMEQUAL", [][]byte{Int64Bytes(0)}},

		{"2MUL 2 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"2MUL 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},
		{"2MUL -2 NUMEQUAL", [][]byte{Int64Bytes(-1)}},

		{"2DIV 1 NUMEQUAL", [][]byte{Int64Bytes(2)}},
		{"2DIV 0 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"2DIV 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},
		{"2DIV 0 NUMEQUAL", [][]byte{Int64Bytes(-1)}},
		{"2DIV -1 NUMEQUAL", [][]byte{Int64Bytes(-2)}},

		{"NEGATE -1 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"NEGATE 1 NUMEQUAL", [][]byte{Int64Bytes(-1)}},
		{"NEGATE 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},

		{"ABS 1 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"ABS 1 NUMEQUAL", [][]byte{Int64Bytes(-1)}},
		{"ABS 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},

		{"0NOTEQUAL", [][]byte{Int64Bytes(1)}},
		{"0NOTEQUAL NOT", [][]byte{Int64Bytes(0)}},

		{"ADD 5 NUMEQUAL", [][]byte{Int64Bytes(2), Int64Bytes(3)}},

		{"SUB 2 NUMEQUAL", [][]byte{Int64Bytes(5), Int64Bytes(3)}},

		{"MUL 6 NUMEQUAL", [][]byte{Int64Bytes(2), Int64Bytes(3)}},

		{"DIV 2 NUMEQUAL", [][]byte{Int64Bytes(6), Int64Bytes(3)}},

		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(6), Int64Bytes(2)}},
		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(-6), Int64Bytes(2)}},
		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(6), Int64Bytes(-2)}},
		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(-6), Int64Bytes(-2)}},
		{"MOD 2 NUMEQUAL", [][]byte{Int64Bytes(12), Int64Bytes(10)}},
		{"MOD 8 NUMEQUAL", [][]byte{Int64Bytes(-12), Int64Bytes(10)}},
		{"MOD -8 NUMEQUAL", [][]byte{Int64Bytes(12), Int64Bytes(-10)}},
		{"MOD -2 NUMEQUAL", [][]byte{Int64Bytes(-12), Int64Bytes(-10)}},

		{"LSHIFT 2 NUMEQUAL", [][]byte{Int64Bytes(1), Int64Bytes(1)}},
		{"LSHIFT 4 NUMEQUAL", [][]byte{Int64Bytes(1), Int64Bytes(2)}},
		{"LSHIFT -2 NUMEQUAL", [][]byte{Int64Bytes(-1), Int64Bytes(1)}},
		{"LSHIFT -4 NUMEQUAL", [][]byte{Int64Bytes(-1), Int64Bytes(2)}},

		{"1 1 BOOLAND", nil},
		{"1 0 BOOLAND NOT", nil},
		{"0 1 BOOLAND NOT", nil},
		{"0 0 BOOLAND NOT", nil},

		{"1 1 BOOLOR", nil},
		{"1 0 BOOLOR", nil},
		{"0 1 BOOLOR", nil},
		{"0 0 BOOLOR NOT", nil},

		{"1 2 OR 3 EQUAL", nil},

		// control ops
		{"TRUE FALSE IF FAIL ENDIF", nil},
		{"FALSE IF FAIL ELSE TRUE ENDIF", nil},
		{"TRUE TRUE NOTIF FAIL ENDIF", nil},
		{"TRUE NOTIF FAIL ELSE TRUE ENDIF", nil},
		{"17 FALSE TRUE TRUE TRUE WHILE DROP ENDWHILE 17 NUMEQUAL", nil},

		// splice ops
		{"0 CATPUSHDATA 0x0000 EQUAL", [][]byte{{0x00}}},
		{"0 0xff CATPUSHDATA 0x01ff EQUAL", nil},
		{"CATPUSHDATA 0x050105 EQUAL", [][]byte{{0x05}, {0x05}}},
		{"CATPUSHDATA 0xff01ff EQUAL", [][]byte{{0xff}, {0xff}}},
		{"0 0xcccccc CATPUSHDATA 0x03cccccc EQUAL", nil},
		{"0x05 0x05 SWAP 0xdeadbeef CATPUSHDATA DROP 0x05 EQUAL", nil},
		{"0x05 0x05 SWAP 0xdeadbeef CATPUSHDATA DROP 0x05 EQUAL", nil},

		// control flow ops
		{"1 IF 1 ENDIF", nil},
		{"1 DUP IF ENDIF", nil},
		{"1 DUP IF ELSE ENDIF", nil},
		{"1 IF 1 ELSE ENDIF", nil},
		{"0 IF ELSE 1 ENDIF", nil},

		{"1 1 IF IF 1 ELSE 0 ENDIF ENDIF", nil},
		{"1 0 IF IF 1 ELSE 0 ENDIF ENDIF", nil},
		{"1 1 IF IF 1 ELSE 0 ENDIF ELSE IF 0 ELSE 1 ENDIF ENDIF", nil},
		{"0 0 IF IF 1 ELSE 0 ENDIF ELSE IF 0 ELSE 1 ENDIF ENDIF", nil},
		{"0 IF 1 IF FAIL ELSE FAIL ENDIF ELSE 1 ENDIF", nil},
		{"1 IF 1 ELSE 1 IF FAIL ELSE FAIL ENDIF ENDIF", nil},

		{"1 0 NOTIF IF 1 ELSE 0 ENDIF ENDIF", nil},
		{"1 1 NOTIF IF 1 ELSE 0 ENDIF ENDIF", nil},
		{"1 0 NOTIF IF 1 ELSE 0 ENDIF ELSE IF 0 ELSE 1 ENDIF ENDIF", nil},
		{"0 1 NOTIF IF 1 ELSE 0 ENDIF ELSE IF 0 ELSE 1 ENDIF ENDIF", nil},

		{"1 WHILE 0 ENDWHILE", nil},
		{"1 WHILE NOT ENDWHILE 1", nil},
		{"1 WHILE IF 0 ELSE 1 ENDIF ENDWHILE 1", nil},
		{"0 WHILE 0 ENDWHILE 1", nil},
		{"1 WHILE IF 1 0 ELSE 1 ENDIF ENDWHILE", nil},
		{"0 1 WHILE DROP 1ADD DUP 10 LESSTHAN ENDWHILE 10 NUMEQUAL", nil},
		{"0 1 2 3 4 5 6 WHILE DROP ENDWHILE 1", nil},
		{"0 1 WHILE DROP 1ADD TOALTSTACK 0 ENDWHILE FROMALTSTACK", nil},
		{"1 WHILE WHILE 0 ENDWHILE 0 ENDWHILE", nil},
	}
	for i, c := range cases {
		progSrc := c.prog
		if !expectOK {
			progSrc += " NOT"
		}
		prog, err := Compile(progSrc)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("* case %d, prog [%s] [%x]\n", i, progSrc, prog)
		trace := new(tracebuf)
		TraceOut = trace
		vm := &virtualMachine{
			program:   prog,
			runLimit:  initialRunLimit,
			dataStack: append([][]byte{}, c.args...),
		}
		ok, err := vm.run()
		if err == nil {
			if ok != expectOK {
				trace.dump()
				t.Errorf("case %d [%s]: expected %v result, got %v", i, progSrc, expectOK, ok)
			}
		} else {
			trace.dump()
			t.Errorf("case %d [%s]: unexpected error: %s", i, progSrc, err)
		}
		if testing.Verbose() && (ok == expectOK) && err == nil {
			trace.dump()
			fmt.Println("")
		}
	}
}
