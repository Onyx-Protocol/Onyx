package raft

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var idCases = []struct {
	id   uint64
	data []byte
}{
	{1, []byte{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31, 0x89}},
	{2, []byte{0, 0, 0, 0, 0, 0, 0, 2, 0x6d, 0x13, 0xc2, 0x7d}},
}

var idErrorCases = []struct {
	data []byte
}{
	{[]byte{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31, 0x89, 0}}, //add extra byte
	{[]byte{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31}},          //missing byte
	{[]byte{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31, 0x0}},     //bad crc
	{[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x8c, 0x28, 0xb2, 0x8a}},    //bad id
}

func TestWriteID(t *testing.T) {
	dir, err := ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range idCases {
		err = writeID(dir, test.id)
		if err != nil {
			t.Error(err)
			continue
		}
		got, err := ioutil.ReadFile(filepath.Join(dir, "id"))
		if err != nil {
			t.Error(err)
			continue
		}
		if !bytes.Equal(got, test.data) {
			t.Errorf("writeID(%d) => %x want %x", test.id, got, test.data)
		}
	}
}

func TestReadID(t *testing.T) {
	dir, err := ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range idCases {
		err = ioutil.WriteFile(filepath.Join(dir, "id"), test.data, 0666)
		if err != nil {
			t.Error(err)
			continue
		}

		got, err := readID(dir)
		if err != nil {
			t.Error(err)
			continue
		}
		if got != test.id {
			t.Errorf("readID() => %d want %d", got, test.id)
		}
	}
}

func TestReadIDError(t *testing.T) {
	dir, err := ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range idErrorCases {
		err = ioutil.WriteFile(filepath.Join(dir, "id"), test.data, 0666)
		if err != nil {
			t.Error(err)
			continue
		}

		_, err := readID(dir)
		if err == nil {
			t.Errorf("readID of %v => err = nil, want error", test.data)
		}
	}
}
