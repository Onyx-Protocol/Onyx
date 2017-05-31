package compiler

import "strings"

var optimizations = []struct {
	before, after string
}{
	{"0 ROLL", ""},
	{"0 PICK", "DUP"},
	{"1 ROLL", "SWAP"},
	{"1 PICK", "OVER"},
	{"2 ROLL", "ROT"},
	{"TRUE VERIFY", ""},
	{"SWAP SWAP", ""},
	{"OVER OVER", "2DUP"},
	{"SWAP OVER", "TUCK"},
	{"DROP DROP", "2DROP"},
	{"SWAP DROP", "NIP"},
	{"5 ROLL 5 ROLL", "2ROT"},
	{"3 PICK 3 PICK", "2OVER"},
	{"3 ROLL 3 ROLL", "2SWAP"},
	{"2 PICK 2 PICK 2 PICK", "3DUP"},
	{"1 ADD", "1ADD"},
	{"1 SUB", "1SUB"},
	{"EQUAL VERIFY", "EQUALVERIFY"},
	{"SWAP TXSIGHASH ROT", "TXSIGHASH SWAP"},
	{"SWAP EQUAL", "EQUAL"},
	{"SWAP EQUALVERIFY", "EQUALVERIFY"},
	{"SWAP ADD", "ADD"},
	{"SWAP BOOLAND", "BOOLAND"},
	{"SWAP BOOLOR", "BOOLOR"},
	{"SWAP MIN", "MIN"},
	{"SWAP MAX", "MAX"},
	{"DUP 2 PICK EQUAL", "2DUP EQUAL"},
	{"DUP 2 PICK EQUALVERIFY", "2DUP EQUALVERIFY"},
	{"DUP 2 PICK ADD", "2DUP ADD"},
	{"DUP 2 PICK BOOLAND", "2DUP BOOLAND"},
	{"DUP 2 PICK BOOLOR", "2DUP BOOLOR"},
	{"DUP 2 PICK MIN", "2DUP MIN"},
	{"DUP 2 PICK MAX", "2DUP MAX"},
}

func optimize(opcodes string) string {
	opcodes = " " + opcodes + " "
	looping := true
	for looping {
		looping = false
		for _, o := range optimizations {
			before := " " + o.before + " "
			var after string
			if o.after == "" {
				after = " "
			} else {
				after = " " + o.after + " "
			}
			newOpcodes := strings.Replace(opcodes, before, after, -1)
			if newOpcodes != opcodes {
				looping = true
				opcodes = newOpcodes
			}
		}
	}
	return strings.TrimSpace(opcodes)
}
