package vm

import "encoding/binary"

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
	res := make([]byte, 8)
	// converting int64 to uint64 is a safe operation that
	// preserves all data
	binary.LittleEndian.PutUint64(res, uint64(n))
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

	res := binary.LittleEndian.Uint64(padded[:])
	// converting uint64 to int64 is a safe operation that
	// preserves all data
	return int64(res), nil
}
