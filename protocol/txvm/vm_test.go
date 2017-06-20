package txvm

import (
	"testing"
)

func opTracer(t testing.TB) func(stack, byte, []byte, []byte) {
	return func(s stack, op byte, data, p []byte) {
		if op >= BaseData {
			t.Logf("[%x]\t\t#stack len: %d", data, s.Len())
		} else if op >= MinInt {
			t.Logf("%d\t\t#stack len: %d", op-MinInt, s.Len())
		} else {
			t.Logf("%s\t\t#stack len: %d", OpNames[op], s.Len())
		}
	}
}

func TestTx(t *testing.T) {
	proof, err := Assemble(`
		""x 10000 0 [1] 4 tuple anchor
		10 ""x [1] ""x 3 tuple issue
		[1] 1 ""x lock
		satisfy satisfy
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
		Nonce: []ID{
			{},
		},
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}
