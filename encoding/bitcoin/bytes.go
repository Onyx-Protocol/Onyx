package bitcoin

import (
	"io"

	"chain/errors"
)

// ErrSize indicates the size of a string or []byte being read
// is greater than the limit given to ReadString or ReadBytes.
var ErrSize = errors.New("bitcoin string too big")

const maxInt = uint64(int(^uint(0) >> 1))

// ReadString reads a variable length string from r
// and returns it as a Go string.
// A Bitcoin string is encoded as a varint
// containing the length of the string
// followed by the bytes that represent the string itself.
// If the size of the string is greater than maxSize,
// ReadString will not read the payload
// and will return ErrSize.
func ReadString(r io.Reader, maxSize int) (string, error) {
	n, err := ReadVarint(r)
	if err != nil {
		return "", err
	}
	if n > maxInt || int(n) > maxSize {
		return "", ErrSize
	}

	buf := make([]byte, n)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// WriteString serializes s to w
// as a varint containing the length of the string
// followed by the bytes that represent the string itself.
// It returns the total number of bytes written.
func WriteString(w io.Writer, s string) (int, error) {
	ew := errors.NewWriter(w)
	WriteVarint(ew, uint64(len(s)))
	io.WriteString(ew, s)
	return int(ew.Written()), ew.Err()
}

// ReadBytes reads a variable length byte array.
// A byte array is encoded as a varint
// containing the length of the array
// followed by the bytes themselves.
// If the size of the data is greater than maxSize,
// ReadBytes will not read the payload
// and will return ErrSize.
func ReadBytes(r io.Reader, maxSize int) ([]byte, error) {
	n, err := ReadVarint(r)
	if err != nil {
		return nil, err
	}
	if n > maxInt || int(n) > maxSize {
		return nil, ErrSize
	}

	b := make([]byte, n)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// WriteBytes writes p as a varint
// containing the number of bytes,
// followed by the bytes themselves.
// It returns the total number of bytes written.
func WriteBytes(w io.Writer, p []byte) (int, error) {
	ew := errors.NewWriter(w)
	WriteVarint(ew, uint64(len(p)))
	ew.Write(p)
	return int(ew.Written()), ew.Err()
}
