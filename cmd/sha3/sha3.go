package main

import (
	"io"
	"os"

	"golang.org/x/crypto/sha3"
)

// Usage:
//   sha3 <bytes >hash

func main() {
	h := sha3.New256()
	_, err := io.Copy(h, os.Stdin)
	if err != nil {
		panic(err)
	}
	_, err = os.Stdout.Write(h.Sum(nil))
	if err != nil {
		panic(err)
	}
}
