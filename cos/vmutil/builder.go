package vmutil

import "chain/cos/vm"

type Builder struct {
	Program []byte
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) AddInt64(n int64) *Builder {
	b.Program = append(b.Program, vm.PushdataInt64(n)...)
	return b
}

func (b *Builder) AddData(data []byte) *Builder {
	b.Program = append(b.Program, vm.PushdataBytes(data)...)
	return b
}

func (b *Builder) AddOp(op vm.Op) *Builder {
	b.Program = append(b.Program, byte(op))
	return b
}
