package bitcoin

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

const testMaxSize = 32 * 1024 * 1024

func TestStringOk(t *testing.T) {
	// str256 is a string that takes a 2-byte varint to encode.
	str256 := strings.Repeat("test", 64)

	tests := []struct {
		decoded string
		encoded []byte
	}{
		{"", []byte{0}},
		{"Test", []byte{0x04, 'T', 'e', 's', 't'}},
		{str256, append([]byte{0xfd, 0x00, 0x01}, str256...)}, // 3-byte varint + string
	}

	for _, test := range tests {
		var buf bytes.Buffer
		_, err := WriteString(&buf, test.decoded)
		if err != nil {
			t.Errorf("WriteString(%q) unexpected error %v", test.decoded, err)
		}
		if got := buf.Bytes(); !bytes.Equal(got, test.encoded) {
			t.Errorf("WriteString(%q) = [%x] want [%x]", test.decoded, got, test.encoded)
		}

		got, err := ReadString(bytes.NewReader(test.encoded), testMaxSize)
		if err != nil {
			t.Errorf("ReadString(%x) unexpected err = %v", test.encoded, err)
		}
		if got != test.decoded {
			t.Errorf("ReadString(%x) = %q want %q", test.encoded, got, test.decoded)
		}
	}
}

func TestStringErr(t *testing.T) {
	// str256 is a string that takes a 2-byte varint to encode.
	str256 := strings.Repeat("test", 64)

	tests := []struct {
		decoded  string // Value to encode
		encoded  []byte // Wire encoding
		max      int    // Max size of fixed buffer to induce errors
		writeErr error  // Expected write error
		readErr  error  // Expected read error
	}{
		// Latest protocol version with intentional read/write errors.
		// Force errors on empty string.
		{"", []byte{}, 0, io.ErrShortWrite, io.EOF},
		// Force error on single byte varint + string.
		{"Test", []byte{0x04, 'a'}, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
		// Force errors on 2-byte varint + string.
		{str256, []byte{0xfd, 0x01}, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
	}

	for _, test := range tests {
		_, err := WriteString(newBuffer(test.max), test.decoded)
		if err != test.writeErr {
			t.Errorf("WriteString(%q) err = %v want %v", test.decoded, err, test.writeErr)
		}

		_, err = ReadString(bytes.NewReader(test.encoded), testMaxSize)
		if err != test.readErr {
			t.Errorf("ReadString(%x) err = %v want %v", test.encoded, err, test.readErr)
		}
	}
}

func TestReadStringErr(t *testing.T) {
	cases := [][]byte{
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}

	for _, test := range cases {
		_, err := ReadString(bytes.NewReader(test), testMaxSize)
		if err == nil {
			t.Errorf("ReadString(%x) err = nil want error", test)
		}
	}
}

func TestBytesOk(t *testing.T) {
	// bytes256 is a byte array that takes a 2-byte varint to encode.
	bytes256 := bytes.Repeat([]byte{0x01}, 256)

	tests := []struct {
		decoded []byte
		encoded []byte
	}{
		{[]byte{}, []byte{0}},                                     // Empty byte array
		{[]byte{0x01}, []byte{0x01, 0x01}},                        // Single byte varint + byte array
		{bytes256, append([]byte{0xfd, 0x00, 0x01}, bytes256...)}, // 2-byte varint + byte array
	}

	for _, test := range tests {
		var buf bytes.Buffer
		_, err := WriteBytes(&buf, test.decoded)
		if err != nil {
			t.Errorf("WriteBytes(%x) error %v", test.decoded, err)
		}
		if got := buf.Bytes(); !bytes.Equal(got, test.encoded) {
			t.Errorf("WriteBytes(%x) = [%x] want [%x]", test.decoded, got, test.encoded)
		}

		got, err := ReadBytes(bytes.NewReader(test.encoded), testMaxSize)
		if err != nil {
			t.Errorf("ReadBytes(%x) unexpected error %v", test.encoded, err)
		}
		if !bytes.Equal(got, test.decoded) {
			t.Errorf("ReadBytes(%x) = [%x] want [%x]", test.decoded, got, test.encoded)
		}
	}
}

func TestBytesErr(t *testing.T) {
	// bytes256 is a byte array that takes a 2-byte varint to encode.
	bytes256 := bytes.Repeat([]byte{0x01}, 256)

	tests := []struct {
		decoded  []byte // Byte Array to write
		encoded  []byte // Wire encoding
		max      int    // Max size of fixed buffer to induce errors
		writeErr error  // Expected write error
		readErr  error  // Expected read error
	}{
		// Latest protocol version with intentional read/write errors.
		// Force errors on empty byte array.
		{[]byte{}, []byte{}, 0, io.ErrShortWrite, io.EOF},
		// Force error on single byte varint + byte array.
		{[]byte{0x01, 0x02, 0x03}, []byte{0x04, 'a'}, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
		// Force errors on 2-byte varint + byte array.
		{bytes256, []byte{0xfd, 0x01}, 2, io.ErrShortWrite, io.ErrUnexpectedEOF},
	}

	for _, test := range tests {
		w := newBuffer(test.max)
		_, err := WriteBytes(w, test.decoded)
		if err != test.writeErr {
			t.Errorf("WriteBytes(%x) err = %v want %v", test.decoded, err, test.writeErr)
		}

		_, err = ReadBytes(bytes.NewReader(test.encoded), testMaxSize)
		if err != test.readErr {
			t.Errorf("ReadBytes(%x) err = %v want %v", test.encoded, err, test.readErr)
		}
	}
}

func TestReadBytesErr(t *testing.T) {
	cases := [][]byte{
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}

	for _, test := range cases {
		_, err := ReadBytes(bytes.NewReader(test), testMaxSize)
		if err == nil {
			t.Errorf("ReadString(%x) err = nil want error", test)
		}
	}
}
