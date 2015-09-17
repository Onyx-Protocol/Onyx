package bitcoin

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

func TestVarintOk(t *testing.T) {
	tests := []struct {
		decoded uint64
		encoded []byte
	}{
		{0, []byte{0x00}},                                  // Single byte
		{0xfc, []byte{0xfc}},                               // Max single byte
		{0xfd, []byte{0xfd, 0x0fd, 0x00}},                  // Min 2-byte
		{0xffff, []byte{0xfd, 0xff, 0xff}},                 // Max 2-byte
		{0x10000, []byte{0xfe, 0x00, 0x00, 0x01, 0x00}},    // Min 4-byte
		{0xffffffff, []byte{0xfe, 0xff, 0xff, 0xff, 0xff}}, // Max 4-byte
		// Min 8-byte
		{0x100000000, []byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}},
		// Max 8-byte
		{0xffffffffffffffff, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		_, err := WriteVarint(&buf, test.decoded)
		if err != nil {
			t.Errorf("WriteVarint(%#x) unexpected err %v", test.decoded, err)
		}
		if got := buf.Bytes(); !bytes.Equal(got, test.encoded) {
			t.Errorf("WriteVarint(%#x) = %x want %x", test.decoded, got, test.encoded)
		}

		rbuf := bytes.NewReader(test.encoded)
		got, err := ReadVarint(rbuf)
		if err != nil {
			t.Errorf("ReadVarint(%x) unexpected err %v", test.encoded, err)
		}
		if got != test.decoded {
			t.Errorf("ReadVarint(%x) = %#x want %#x", test.encoded, got, test.decoded)
		}
	}
}

func TestVarintErr(t *testing.T) {
	tests := []struct {
		decoded  uint64 // Value to encode
		encoded  []byte // Wire encoding
		max      int    // Max size of fixed buffer to induce errors
		writeErr error  // Expected write error
		readErr  error  // Expected read error
	}{
		// Force errors on discriminant.
		{0, []byte{}, 0, io.ErrShortWrite, io.EOF},
		// Force errors on 2-byte read/write.
		{0xfd, []byte{0xfd, 0x01}, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
		// Force errors on 4-byte read/write.
		{0x10000, []byte{0xfe, 0x01}, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
		// Force errors on 8-byte read/write.
		{0x100000000, []byte{0xff, 0x01}, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
	}

	for _, test := range tests {
		w := newBuffer(test.max)
		_, err := WriteVarint(w, test.decoded)
		if err != test.writeErr {
			t.Errorf("WriteVarint(%#x) err = %v want %v", test.decoded, err, test.writeErr)
		}

		_, err = ReadVarint(bytes.NewReader(test.encoded))
		if err != test.readErr {
			t.Errorf("ReadVarint(%x) err = %v want %v", test.encoded, err, test.readErr)
		}
	}
}

func TestReadVarintErr(t *testing.T) {
	tests := []struct {
		v    uint64 // Value to get the serialized size for
		size int    // Expected serialized size
	}{
		{0, 1},                  // Single byte
		{0xfc, 1},               // Max single byte
		{0xfd, 3},               // Min 2-byte
		{0xffff, 3},             // Max 2-byte
		{0x10000, 5},            // Min 4-byte
		{0xffffffff, 5},         // Max 4-byte
		{0x100000000, 9},        // Min 8-byte
		{0xffffffffffffffff, 9}, // Max 8-byte
	}

	for _, test := range tests {
		got, _ := WriteVarint(ioutil.Discard, test.v)
		if got != test.size {
			t.Errorf("WriteVarint(%#x) n = %d want %d", test.v, got, test.size)
		}
	}
}

// buffer writes to its internal buffer until full,
// then returns an error.
type buffer struct {
	b []byte
	w int
}

func (b *buffer) Write(p []byte) (n int, err error) {
	n = copy(b.b[b.w:], p)
	b.w += n
	if n < len(p) {
		err = io.ErrShortWrite
	}
	return
}

func newBuffer(size int) *buffer {
	return &buffer{b: make([]byte, size)}
}
