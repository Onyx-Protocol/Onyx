package main

import (
	"io/ioutil"
	"os"

	"golang.org/x/crypto/sha3"
)

func main() {
	b, err := ioutil.ReadAll(os.Stdin)
	h := sha3.Sum256(b)
	os.Stdout.Write(h[:])
}
