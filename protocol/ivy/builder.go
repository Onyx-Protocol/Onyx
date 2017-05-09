package ivy

import (
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

// builder is just like vmutil.Builder but it holds back any OP_VERIFY
// instructions unless/until something is added after it. An OP_VERIFY
// added at the end is left off entirely.
type builder struct {
	b             *vmutil.Builder
	verifyPending bool
}

func newBuilder() *builder {
	return &builder{b: vmutil.NewBuilder(true)}
}

func (b *builder) addInt64(n int64) *builder {
	b.resolve()
	b.b.AddInt64(n)
	return b
}

func (b *builder) addData(data []byte) *builder {
	b.resolve()
	b.b.AddData(data)
	return b
}

func (b *builder) addRawBytes(data []byte) *builder {
	b.resolve()
	b.b.AddRawBytes(data)
	return b
}

func (b *builder) addOp(op vm.Op) *builder {
	b.resolve()
	if op == vm.OP_VERIFY {
		b.verifyPending = true
	} else {
		b.b.AddOp(op)
	}
	return b
}

func (b *builder) newJumpTarget() int {
	return b.b.NewJumpTarget()
}

func (b *builder) setJumpTarget(target int) *builder {
	b.b.SetJumpTarget(target)
	return b
}

func (b *builder) addJump(target int) *builder {
	b.b.AddJump(target)
	return b
}

func (b *builder) addJumpIf(target int) *builder {
	b.b.AddJumpIf(target)
	return b
}

func (b *builder) build() ([]byte, error) {
	return b.b.Build()
}

func (b *builder) resolve() {
	if b.verifyPending {
		b.b.AddOp(vm.OP_VERIFY)
		b.verifyPending = false
	}
}
