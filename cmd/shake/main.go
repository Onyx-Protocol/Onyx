package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/crypto/sha3"
)

func main() {
	size := flag.Int("n", 256, "hash state size in `bits`: 128 or 256")
	flag.Parse()

	var h sha3.ShakeHash
	switch *size {
	case 128:
		h = sha3.NewShake128()
	case 256:
		h = sha3.NewShake256()
	default:
		fmt.Fprintf(os.Stderr, "unsupported hash size %d (must be 128 or 256)", *size)
		os.Exit(2)
	}

	_, err := io.Copy(h, os.Stdin)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = io.Copy(os.Stdout, h)
	if err != nil {
		log.Fatalln(err)
	}
}
