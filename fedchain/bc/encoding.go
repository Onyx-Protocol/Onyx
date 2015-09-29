package bc

import (
	"encoding/binary"
	"io"

	"chain/errors"
	chainio "chain/io"
)

// endianness is the default endian encoding (little or big)
var endianness = binary.BigEndian

func writeUvarint(w *errors.Writer, x uint64) {
	var buf [9]byte
	n := binary.PutUvarint(buf[:], x)
	w.Write(buf[0:n])
}

func writeBytes(w *errors.Writer, data []byte) {
	writeUvarint(w, uint64(len(data)))
	w.Write(data)
}

func readBytes(r *errors.Reader, b *[]byte) {
	n := readUvarint(r)
	if n < 1 {
		return
	}
	*b = make([]byte, n)
	io.ReadFull(r, *b)
}

func readUvarint(r *errors.Reader) uint64 {
	n, err := binary.ReadUvarint(chainio.ByteReader(r))
	if err != nil {
		r.Err = err
	}
	return n
}
