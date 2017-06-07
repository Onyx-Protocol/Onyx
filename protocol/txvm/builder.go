package txvm

type Builder struct {
	program []byte
}

func (b *Builder) Op(code byte) *Builder {
	b.program = append(b.program, code)
	return b
}

func (b *Builder) Data(data []byte) *Builder {
	b.program = append(b.program, pushData(data)...)
	return b
}

func (b *Builder) Int64(n int64) *Builder {
	b.program = append(b.program, pushInt64(n)...)
	return b
}

func (b *Builder) Build() []byte {
	return b.program
}
