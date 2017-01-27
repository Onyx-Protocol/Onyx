package bc

import "io"

// OutputID identifies previous transaction output in transaction inputs.
type OutputID struct{ Hash }

// WriteTo writes p to w.
func (outid *OutputID) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(outid.Hash[:])
	return int64(n), err
}

func (outid *OutputID) readFrom(r io.Reader) (int, error) {
	return io.ReadFull(r, outid.Hash[:])
}
