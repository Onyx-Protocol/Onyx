package net

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestDialUnix(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "lookup-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempdir)
	path := filepath.Join(tempdir, "sock")
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	go l.Accept()

	d := &Dialer{}
	c, err := d.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}
