package rotation

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestRotate(t *testing.T) {
	defer os.Remove("x")
	defer os.Remove("x.1")
	defer os.Remove("x.2")
	defer os.Remove("x.3") // just in case
	os.Remove("x")
	os.Remove("x.1")
	os.Remove("x.2")
	os.Remove("x.3") // just in case

	f := Create("x", 1e6, 2)

	touch("x")
	f.rotate()
	if !isRegular("x.1") {
		t.Fatal("want rotated file x.1")
	}
	if isRegular("x.2") {
		t.Fatal("want no rotated file x.2")
	}
	if isRegular("x.3") {
		t.Fatal("want no rotated file x.3")
	}

	touch("x")
	f.rotate()
	if !isRegular("x.1") {
		t.Fatal("want rotated file x.1")
	}
	if !isRegular("x.2") {
		t.Fatal("want rotated file x.2")
	}
	if isRegular("x.3") {
		t.Fatal("want no rotated file x.3")
	}

	touch("x")
	f.rotate()
	if !isRegular("x.1") {
		t.Fatal("want rotated file x.1")
	}
	if !isRegular("x.2") {
		t.Fatal("want rotated file x.2")
	}
	if isRegular("x.3") {
		t.Fatal("want no rotated file x.3")
	}
}

func TestRotate0(t *testing.T) {
	defer os.Remove("x")
	defer os.Remove("x.1")
	defer os.Remove("x.2") // just in case
	os.Remove("x")
	os.Remove("x.1")
	os.Remove("x.2") // just in case

	f := Create("x", 1e6, 0)

	touch("x")
	f.rotate()
	if !isRegular("x.1") {
		t.Fatal("want rotated file x.1")
	}
	if isRegular("x.2") {
		t.Fatal("want no rotated file x.2")
	}

	touch("x")
	f.rotate()
	if !isRegular("x.1") {
		t.Fatal("want rotated file x.1")
	}
	if isRegular("x.2") {
		t.Fatal("want no rotated file x.2")
	}
}

func TestInternalWriteNoRotate(t *testing.T) {
	defer os.Remove("x")
	defer os.Remove("x.1") // just in case
	os.Remove("x")
	os.Remove("x.1") // just in case

	f := &File{
		base: "x",
		n:    1,
		size: 10,
		w:    0,
	}
	b := []byte("abc\n")
	n, err := f.write(b)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(b) {
		t.Fatalf("write(%q) = %d want %d", b, n, len(b))
	}
	if isRegular("x.1") {
		t.Fatal("want no rotated file x.1")
	}
	if f.w != int64(len(b)) {
		t.Fatalf("f.w = %d want %d", f.w, len(b))
	}
}

func TestInternalWriteRotate(t *testing.T) {
	defer os.Remove("x")
	defer os.Remove("x.1")
	os.Remove("x")
	os.Remove("x.1")

	f0, err := os.Create("x")
	if err != nil {
		t.Fatal(err)
	}

	f := &File{
		base: "x",
		n:    1,
		size: 10,
		w:    9,
		f:    f0,
	}
	b := []byte("abc\n")
	n, err := f.write(b)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(b) {
		t.Fatalf("write(%q) = %d want %d", b, n, 4)
	}
	if !isRegular("x.1") {
		t.Fatal("want rotated file x.1")
	}
	if f.f == f0 {
		t.Fatal("want new file object")
	}
	if f.w != int64(len(b)) {
		t.Fatalf("f.w = %d want %d", f.w, len(b))
	}
}

func TestWrite(t *testing.T) {
	defer os.Remove("x")
	defer os.Remove("x.1")
	os.Remove("x")
	os.Remove("x.1")
	f := Create("x", 1e6, 1)

	b0 := []byte("ab")
	n, err := f.Write(b0)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(b0) {
		t.Fatalf("n = %d want %d", n, len(b0))
	}
	if !bytes.Equal(f.buf, b0) {
		t.Fatalf("buf = %q want %q", f.buf, b0)
	}
	if f.w != 0 {
		t.Fatalf("w = %d want 0", f.w)
	}

	b1 := []byte("c\n")
	n, err = f.Write(b1)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(b1) {
		t.Fatalf("n = %d want %d", n, len(b1))
	}
	if !bytes.Equal(f.buf, nil) {
		t.Fatalf("buf = %q want %q", f.buf, "")
	}
	if f.w != int64(len(b0)+len(b1)) {
		t.Fatalf("w = %d want %d", f.w, len(b0)+len(b1))
	}
}

func TestOpenFail(t *testing.T) {
	f := Create("/tmp/improbable-test-path/second-level/x", 1e6, 1)
	b := []byte("abc\nd")
	want := append(dropmsg, 'd')
	n, err := f.Write(b)
	if err == nil {
		t.Error("want error")
	}
	if n != len(b) {
		t.Errorf("n = %d want %d", n, len(b))
	}
	if !bytes.Equal(f.buf, want) {
		t.Fatalf("buf = %q want %q", f.buf, want)
	}

	n, err = f.Write(b)
	if err == nil {
		t.Error("want error")
	}
	if n != len(b) {
		t.Errorf("n = %d want %d", n, len(b))
	}
	// don't accumulate multiple dropped-log messages
	if !bytes.Equal(f.buf, want) {
		t.Fatalf("buf = %q want %q", f.buf, want)
	}
}

func TestAppend(t *testing.T) {
	defer os.Remove("x")
	b0 := []byte("abc\n")
	b1 := []byte("def\n")
	err := ioutil.WriteFile("x", b0, 0666)
	if err != nil {
		t.Fatal(err)
	}
	f := Create("x", 100, 1)
	f.Write(b1)
	if want := int64(len(b0) + len(b1)); f.w != want {
		t.Fatalf("w = %d want %d", f.w, want)
	}
}

func isRegular(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.Mode().IsRegular()
}

func touch(name string) {
	f, _ := os.Create(name)
	if f != nil {
		f.Close()
	}
}
