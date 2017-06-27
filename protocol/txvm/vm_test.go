package txvm

import (
	"testing"
)

func opTracer(t testing.TB) func(stack, byte, []byte, []byte) {
	return func(s stack, op byte, data, p []byte) {
		if op >= BaseData {
			t.Logf("[%x]\t\t#stack len: %d", data, s.Len())
		} else if op >= BaseInt {
			t.Logf("%d\t\t#stack len: %d", op-BaseInt, s.Len())
		} else {
			t.Logf("%s\t\t#stack len: %d", OpNames[op], s.Len())
		}
	}
}

func TestIssue(t *testing.T) {
	tx, err := Assemble(`
		{'nonce', [1 verify], 0, 10000} anchor
		100 {'assetdefinition', {}, [1 verify]} issue
		[1 verify] 1 lock
		summarize
	`)
	if err != nil {
		t.Fatal(err)
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}

func TestSpend(t *testing.T) {
	tx, err := Assemble(`
		{
			'output',
			"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"x,
			{{
				'value',
				"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"x,
				100,
				"00112233445566778899aabbccddeeffffeeddccbbaa99887766554433221100"x,
			}},
			[1 verify],
		} unlock
		retire
		summarize
	`)
	if err != nil {
		t.Fatal(err)
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}

func TestEntries(t *testing.T) {
	tx, err := Assemble(`
		{'nonce', [1 verify], 0, 10000} anchor
		100 {'assetdefinition', {}, [1 verify]} issue
		"abba"x 3 id 2 maketuple encode annotate
		45 split merge
		retire
		0 after
		10000 before
		summarize
	`)
	if err != nil {
		t.Fatal(err)
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}
