// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file at
// https://github.com/golang/go/blob/master/LICENSE.

package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	dataDir = "testdata"
	binary  = "testvet.exe"
)

// Run this shell script, but do it in Go so it can be run by "go test".
// 	go build -o testvet
// 	$(GOROOT)/test/errchk ./testvet testdata/*.go testdata/*.s
// 	rm testvet
//
func TestVet(t *testing.T) {
	switch runtime.GOOS {
	case "plan9", "windows":
		// Plan 9 and Windows systems can't be guaranteed to have Perl and so can't run errchk.
		t.Skipf("skipping test; no Perl on %q", runtime.GOOS)
	}

	// go build
	cmd := exec.Command("go", "build", "-o", binary)
	run(cmd, t)

	// defer removal of vet
	defer os.Remove(binary)

	// errchk ./testvet
	gos, err := filepath.Glob(filepath.Join(dataDir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	asms, err := filepath.Glob(filepath.Join(dataDir, "*.s"))
	if err != nil {
		t.Fatal(err)
	}
	files := append(gos, asms...)
	errchk := filepath.Join(runtime.GOROOT(), "test", "errchk")
	flags := []string{"./" + binary}
	cmd = exec.Command(errchk, append(flags, files...)...)
	if !run(cmd, t) {
		t.Fatalf("vet command failed: %s %v %v", errchk, flags, files)
	}
}

func run(c *exec.Cmd, t *testing.T) bool {
	output, err := c.CombinedOutput()
	os.Stderr.Write(output)
	if err != nil {
		t.Fatal(err)
	}
	// Errchk delights by not returning non-zero status if it finds errors, so we look at the output.
	// It prints "BUG" if there is a failure.
	if !c.ProcessState.Success() {
		return false
	}
	return !bytes.Contains(output, []byte("BUG"))
}

// TestTags verifies that the -tags argument controls which files to check.
func TestTags(t *testing.T) {
	// go build
	cmd := exec.Command("go", "build", "-o", binary)
	run(cmd, t)

	// defer removal of vet
	defer os.Remove(binary)

	args := []string{
		"-tags=testtag",
		"-v", // We're going to look at the files it examines.
		"testdata/tagtest",
	}
	cmd = exec.Command("./"+binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	// file1 has testtag and file2 has !testtag.
	if !bytes.Contains(output, []byte(filepath.Join("tagtest", "file1.go"))) {
		t.Error("file1 was excluded, should be included")
	}
	if bytes.Contains(output, []byte(filepath.Join("tagtest", "file2.go"))) {
		t.Error("file2 was included, should be excluded")
	}
}
