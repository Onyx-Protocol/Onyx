package bc

import (
	"encoding/binary"

	"chain/errors"
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
