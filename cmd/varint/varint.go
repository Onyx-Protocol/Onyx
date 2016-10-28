package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("varint: ")

	// strip the "varint" token off the front of all args
	args := os.Args[1:]

	if len(args) == 0 {
		// decode from stdin
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			errorf("could not read from stdin: %s", err)
		}

		n, nbytes := binary.Uvarint(b)
		if nbytes <= 0 {
			errorf("could not parse uvarint")
		}
		fmt.Println(n)
		return
	}

	// encode from args
	if len(args) != 1 {
		errorf("invalid argument count %d; varint must read from stdin or take 1 argument", len(args))
	}

	val, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		errorf("could not parse base 10 uint")
	}

	var buf [10]byte
	n := binary.PutUvarint(buf[:], val)

	_, err = os.Stdout.Write(buf[:n])
	if err != nil {
		errorf("could not write to stdout: %s", err)
	}
}

func errorf(msg string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(msg, args...))
	os.Exit(1)
}
