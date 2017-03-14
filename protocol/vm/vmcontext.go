package vm

type VMContext interface {
	VMVersion() uint64
	Code() []byte
	Arguments() [][]byte

	TXVersion() (txVersion uint64, ok bool)
}
