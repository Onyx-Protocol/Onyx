package main

import (
	"flag"
	"fmt"
	"hash"
	"io"
	"os"

	"golang.org/x/crypto/sha3"
)

func main() {
	size := flag.Int("n", 256, "output size in `bits`: 224, 256, 384, or 512")
	flag.Parse()

	var h hash.Hash

	switch *size {
	case 224:
		h = sha3.New224()
	case 256:
		h = sha3.New256()
	case 384:
		h = sha3.New384()
	case 512:
		h = sha3.New512()
	default:
		panic(fmt.Errorf("unsupported hash size %d (must be 224, 256, 384, or 512)", *size))
	}
	_, err := io.Copy(h, os.Stdin)
	if err != nil {
		panic(err)
	}
	_, err = os.Stdout.Write(h.Sum(nil))
	if err != nil {
		panic(err)
	}
}
