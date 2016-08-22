package vm

import (
	"bytes"
	"encoding/binary"
)

var trueBytes = []byte{1}

func BoolBytes(b bool) (result []byte) {
	if b {
		return trueBytes
	}
	return []byte{}
}

func AsBool(bytes []byte) bool {
	for _, b := range bytes {
		if b != 0 {
			return true
		}
	}
	return false
}

func Int64Bytes(n int64) []byte {
	if n == 0 {
		return []byte{}
	}
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, n)
	res := b.Bytes()
	for len(res) > 0 && res[len(res)-1] == 0 {
		res = res[:len(res)-1]
	}
	return res
}

func AsInt64(b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if len(b) > 8 {
		return 0, ErrBadValue
	}

	var padded [8]byte
	copy(padded[:], b)
	buf := bytes.NewReader(padded[:])

	var res int64
	err := binary.Read(buf, binary.LittleEndian, &res)

	return res, err
}
