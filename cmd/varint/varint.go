package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

var signed bool

func main() {
	log.SetFlags(0)
	log.SetPrefix("varint: ")

	flag.BoolVar(&signed, "s", false, "signed")
	flagNotBoolVar(&signed, "u", true, "unsigned (default)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		// decode from stdin
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			errorf("could not read from stdin: %s", err)
		}

		var nbytes int
		if signed {
			var n int64
			n, nbytes = binary.Varint(b)
			if nbytes <= 0 {
				errorf("could not parse varint")
			}
			_, err = os.Stdout.Write([]byte(strconv.FormatInt(n, 10)))
			if err != nil {
				errorf("could not write to stdout: %s", err)
			}
		} else {
			var n uint64
			n, nbytes = binary.Uvarint(b)
			if nbytes <= 0 {
				errorf("could not parse uvarint")
			}
			_, err = os.Stdout.Write([]byte(strconv.FormatUint(n, 10)))
			if err != nil {
				errorf("could not write to stdout: %s", err)
			}
		}
		return
	}

	// encode from args
	if len(args) != 1 {
		errorf("invalid argument count %d; varint must read from stdin or take 1 argument", len(args))
	}

	val, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		errorf("could not parse base 10 int")
	}

	var (
		buf [10]byte
		n   int
	)
	if signed {
		n = binary.PutVarint(buf[:], val)
	} else {
		n = binary.PutUvarint(buf[:], uint64(val))
	}

	_, err = os.Stdout.Write(buf[:n])
	if err != nil {
		errorf("could not write to stdout: %s", err)
	}
}

func errorf(msg string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(msg, args...))
	os.Exit(1)
}
