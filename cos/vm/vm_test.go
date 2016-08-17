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
	cases := []struct {
		prog string
		args [][]byte
	}{
		{"TRUE", nil},

		// bitwise ops
		{"0x010f INVERT 0xfef0 EQUAL", nil},

		{"0x03 0x06 AND 0x02 EQUAL", nil},
		{"0x03ff 0x06 AND 0x02 EQUAL", nil},

		{"0x03 0x06 OR 0x07 EQUAL", nil},
		{"0x03ff 0x06 OR 0x07ff EQUAL", nil},

		{"0x03 0x06 XOR 0x05 EQUAL", nil},
		{"0x03ff 0x06 XOR 0x05ff EQUAL", nil},

		// numeric and logical ops
		{"1 1ADD 2 NUMEQUAL", nil},
		{"-1 1ADD 0 NUMEQUAL", nil},

		{"2 1SUB 1 NUMEQUAL", nil},
		{"0 1SUB -1 NUMEQUAL", nil},

		{"1 2MUL 2 NUMEQUAL", nil},
		{"0 2MUL 0 NUMEQUAL", nil},
		{"-1 2MUL -2 NUMEQUAL", nil},

		{"2 2DIV 1 NUMEQUAL", nil},
		{"1 2DIV 0 NUMEQUAL", nil},
		{"0 2DIV 0 NUMEQUAL", nil},
		{"-1 2DIV 0 NUMEQUAL", nil},
		{"-2 2DIV -1 NUMEQUAL", nil},

		{"1 NEGATE -1 NUMEQUAL", nil},
		{"-1 NEGATE 1 NUMEQUAL", nil},
		{"0 NEGATE 0 NUMEQUAL", nil},

		{"1 ABS 1 NUMEQUAL", nil},
		{"-1 ABS 1 NUMEQUAL", nil},
		{"0 ABS 0 NUMEQUAL", nil},

		{"1 0NOTEQUAL", nil},
		{"0 0NOTEQUAL NOT", nil},

		{"2 3 ADD 5 NUMEQUAL", nil},

		{"5 3 SUB 2 NUMEQUAL", nil},

		{"2 3 MUL 6 NUMEQUAL", nil},

		{"6 3 DIV 2 NUMEQUAL", nil},

		{"6 2 MOD 0 NUMEQUAL", nil},
		{"-6 2 MOD 0 NUMEQUAL", nil},
		{"6 -2 MOD 0 NUMEQUAL", nil},
		{"-6 -2 MOD 0 NUMEQUAL", nil},
		{"12 10 MOD 2 NUMEQUAL", nil},
		{"-12 10 MOD 8 NUMEQUAL", nil},
		{"12 -10 MOD -8 NUMEQUAL", nil},
		{"-12 -10 MOD -2 NUMEQUAL", nil},

		{"1 1 LSHIFT 2 NUMEQUAL", nil},
		{"1 2 LSHIFT 4 NUMEQUAL", nil},
		{"-1 1 LSHIFT -2 NUMEQUAL", nil},
		{"-1 2 LSHIFT -4 NUMEQUAL", nil},

		{"1 1 BOOLAND", nil},
		{"1 0 BOOLAND NOT", nil},
		{"0 1 BOOLAND NOT", nil},
		{"0 0 BOOLAND NOT", nil},

		{"1 1 BOOLOR", nil},
		{"1 0 BOOLOR", nil},
		{"0 1 BOOLOR", nil},
		{"0 0 BOOLOR NOT", nil},

		{"1 2 OR 3 EQUAL", nil},
	}
	for i, c := range cases {
		prog, err := Compile(c.prog)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("* case %d, prog [%s] [%x]\n", i, c.prog, prog)
		trace := new(tracebuf)
		vm := &virtualMachine{
			program:   prog,
			runLimit:  initialRunLimit,
			dataStack: append([][]byte{}, c.args...),
			traceOut:  trace,
		}
		ok, err := vm.run()
		if err != nil {
			trace.dump()
			t.Errorf("case %d: unexpected error %s", i, err)
		}
		if !ok {
			trace.dump()
			t.Errorf("case %d: expected true result, got false", i)
		}
		if testing.Verbose() && ok && err == nil {
			trace.dump()
		}
		fmt.Println("")
	}
}
