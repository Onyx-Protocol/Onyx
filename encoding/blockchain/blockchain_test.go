package blockchain

import (
	"bytes"
	"io"
	"io/ioutil"
	"math"
	"reflect"
	"testing"
)

func BenchmarkReadVarint31(b *testing.B) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0x01}
	r := bytes.NewReader(data)
	for i := 0; i < b.N; i++ {
		r.Reset(data)
		ReadVarint31(r)
	}
}

func BenchmarkReadVarint63(b *testing.B) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}
	r := bytes.NewReader(data)
	for i := 0; i < b.N; i++ {
		r.Reset(data)
		ReadVarint63(r)
	}
}

func BenchmarkWriteVarint31(b *testing.B) {
	n := uint64(math.MaxInt32)
	for i := 0; i < b.N; i++ {
		WriteVarint31(ioutil.Discard, n)
	}
}

func BenchmarkWriteVarint63(b *testing.B) {
	n := uint64(math.MaxInt64)
	for i := 0; i < b.N; i++ {
		WriteVarint63(ioutil.Discard, n)
	}
}

func TestVarint31(t *testing.T) {
	cases := []struct {
		n       uint64
		want    []byte
		wantErr error
	}{
		{
			n:    0,
			want: []byte{0},
		},
		{
			n:    500,
			want: []byte{0xf4, 0x03},
		},
		{
			n:       math.MaxInt32 + 1,
			wantErr: ErrRange,
		},
	}

	for _, c := range cases {
		b := new(bytes.Buffer)
		n, err := WriteVarint31(b, c.n)
		if c.wantErr != err {
			t.Errorf("WriteVarint31(%d): err %v, want %v", c.n, err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}
		if n != len(c.want) {
			t.Errorf("WriteVarint31(%d): wrote %d byte(s), want %d", c.n, n, len(c.want))
		}
		if !bytes.Equal(c.want, b.Bytes()) {
			t.Errorf("WriteVarint31(%d): got %x, want %x", c.n, b.Bytes(), c.want)
		}
		v, n, err := ReadVarint31(bytes.NewReader(b.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		if n != len(c.want) {
			t.Errorf("ReadVarint31 [c.n = %d] got %d bytes, want %d", c.n, n, len(c.want))
		}
		if uint64(v) != c.n {
			t.Errorf("ReadVarint31 got %d, want %d", v, c.n)
		}
	}
}

func TestVarint63(t *testing.T) {
	cases := []struct {
		n       uint64
		want    []byte
		wantErr error
	}{
		{
			n:    0,
			want: []byte{0},
		},
		{
			n:    500,
			want: []byte{0xf4, 0x03},
		},
		{
			n:    math.MaxInt32 + 1,
			want: []byte{0x80, 0x80, 0x80, 0x80, 0x08},
		},
		{
			n:       math.MaxInt64 + 1,
			wantErr: ErrRange,
		},
	}

	for _, c := range cases {
		b := new(bytes.Buffer)
		n, err := WriteVarint63(b, c.n)
		if c.wantErr != err {
			t.Errorf("WriteVarint63(%d): err %v, want %v", c.n, err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}
		if n != len(c.want) {
			t.Errorf("WriteVarint63(%d): wrote %d byte(s), want %d", c.n, n, len(c.want))
		}
		if !bytes.Equal(c.want, b.Bytes()) {
			t.Errorf("WriteVarint63(%d): got %x, want %x", c.n, b.Bytes(), c.want)
		}
		v, n, err := ReadVarint63(bytes.NewReader(b.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		if n != len(c.want) {
			t.Errorf("ReadVarint63 [c.n = %d] got %d bytes, want %d", c.n, n, len(c.want))
		}
		if uint64(v) != c.n {
			t.Errorf("ReadVarint63 got %d, want %d", v, c.n)
		}
	}
}

func TestVarstring31(t *testing.T) {
	s := []byte{10, 11, 12}
	b := new(bytes.Buffer)
	_, err := WriteVarstr31(b, s)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{3, 10, 11, 12}
	if !bytes.Equal(b.Bytes(), want) {
		t.Errorf("got %x, want %x", b.Bytes(), want)
	}
	s, _, err = ReadVarstr31(bytes.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	want = []byte{10, 11, 12}
	if !bytes.Equal(s, want) {
		t.Errorf("got %x, expected %x", s, want)
	}
}

func TestVarstrList(t *testing.T) {
	for i := 0; i < 4; i++ {
		// make a list of i+1 strs, each with length i+1, each make of repeating byte i
		strs := make([][]byte, 0, i+1)
		for j := 0; j <= i; j++ {
			str := make([]byte, 0, i+1)
			for k := 0; k <= i; k++ {
				str = append(str, byte(i))
			}
			strs = append(strs, str)
		}
		var buf bytes.Buffer
		_, err := WriteVarstrList(&buf, strs)
		if err != nil {
			t.Fatal(err)
		}
		strs2, _, err := ReadVarstrList(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(strs, strs2) {
			t.Errorf("got %v, want %v", strs2, strs)
		}
	}
}

func TestExtensibleString(t *testing.T) {
	for i := 0; i < 4; i++ {
		// make a string of length i+1
		str := make([]byte, 0, i+1)
		for j := 0; j <= i; j++ {
			str = append(str, byte(i))
		}
		var buf bytes.Buffer
		_, err := WriteExtensibleString(&buf, func(w io.Writer) error {
			_, err := w.Write(str)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		var str2 []byte
		b := buf.Bytes()
		_, err = ReadExtensibleString(bytes.NewReader(b), true, func(r io.Reader) error {
			str2, err = ioutil.ReadAll(r)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(str, str2) {
			t.Errorf("got %x, want %x", str2, str)
		}
		_, err = ReadExtensibleString(bytes.NewReader(b[:i]), false, func(r io.Reader) error {
			return nil
		})
		switch err {
		case nil:
			t.Errorf("got no error, want io.EOF")
		case io.EOF, io.ErrUnexpectedEOF:
		default:
			t.Errorf("got error %s, want io.EOF", err)
		}
		_, err = ReadExtensibleString(bytes.NewReader(b), false, func(r io.Reader) error {
			return nil
		})
		if err != nil {
			t.Error(err)
		}
		_, err = ReadExtensibleString(bytes.NewReader(b), true, func(r io.Reader) error {
			return nil
		})
		switch err {
		case nil:
			t.Errorf("got no error, want ErrLeftover")
		case ErrLeftover:
		default:
			t.Errorf("got error %s, want ErrLeftover")
		}
	}
}
