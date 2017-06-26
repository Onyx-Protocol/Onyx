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
	proof, err := Assemble(`
		{"6e6f6e6365"x, {}, [1], 0, 10000} anchor
		100 {"6173736574646566696e6974696f6e"x, {}, [1]} issue
		[1] 1 ""x lock
		10000 0 ""x header
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}

func TestSpend(t *testing.T) {
	proof, err := Assemble(`
		{
			"6f7574707574"x,
			{},
			"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"x,
			{{
				"76616c7565"x,
				{},
				"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"x,
				100,
				"00112233445566778899aabbccddeeffffeeddccbbaa99887766554433221100"x,
			}},
			[1 verify],
		} unlock
		retire
		10000 0 ""x header
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}

func TestEntries(t *testing.T) {
	proof, err := Assemble(`
		{"6e6f6e6365"x, {}, [1 verify], 0, 10000} anchor
		100 {"6173736574646566696e6974696f6e"x, {}, [1 verify]} issue
		45 split merge
		retire
		10000 0 ""x header
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}
