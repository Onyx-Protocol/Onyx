package bc

// Program encapsulates the code and vm version for a virtual machine
// program.
type Program struct {
	VMVersion uint64
	Code      []byte
}
