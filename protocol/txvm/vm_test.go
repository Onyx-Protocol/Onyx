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
		10000 0 [1] 0 tuple "6e6f6e6365"x 5 tuple anchor
		[1] 0 tuple "6173736574646566696e6974696f6e"x 3 tuple issue
		[1] 1 ""x lock
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
		Nonce: [][32]byte{
			{},
		},
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}
