package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"chain/fedchain/bc"
)

const (
	help = `
Usage:

	decode tx [hex-encoded bc.Tx]
	decode block [hex-encoded bc.Block]
	decode blockheader [hex-encoded bc.BlockHeader]
`
)

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func prettyPrint(obj interface{}) {
	j, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fatalf("error json-marshaling: %s", err)
	}
	fmt.Println(string(j))
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, help)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println(strings.TrimSpace(help))
		return
	}

	switch strings.ToLower(args[0]) {
	case "blockheader":
		b, err := hex.DecodeString(args[1])
		if err != nil {
			fatalf("err decoding hex: %s", err)
		}

		var bh bc.BlockHeader
		err = bh.Scan(b)
		if err != nil {
			fatalf("error decoding: %s", err)
		}

		// The struct doesn't have the hash, so calculate it and print it
		// before pretty printing the header.
		fmt.Printf("Block Hash: %s\n", bh.Hash())
		prettyPrint(bh)
	case "block":
		b, err := hex.DecodeString(args[1])
		if err != nil {
			fatalf("err decoding hex: %s", err)
		}

		var block bc.Block
		err = block.Scan(b)
		if err != nil {
			fatalf("error decoding: %s", err)
		}

		// The struct doesn't have the hash, so calculate it and print it
		// before pretty printing the block
		fmt.Printf("Block Hash: %s\n", block.Hash())
		prettyPrint(block)
	case "tx":
		var tx bc.Tx
		err := tx.UnmarshalText([]byte(args[1]))
		if err != nil {
			fatalf("error decoding: %s", err)
		}
		prettyPrint(tx)
	default:
		fatalf("unrecognized entity `%s`", args[0])
	}
}
