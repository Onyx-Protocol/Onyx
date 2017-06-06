// Command decode reads hex-encoded Chain data structures and prints
// the decoded data structures.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"chain/protocol/bc/legacy"
	"chain/protocol/txvm"
	"chain/protocol/vm"
)

const help = `
Command decode reads a data item from stdin, decodes it,
and prints its JSON representation to stdout.

On Mac OS X, to decode an item from the pasteboard,

	pbpaste|decode tx
	pbpaste|decode block
	pbpaste|decode blockheader
	pbpaste|decode script
	pbpaste|decode txvm
`

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
	if len(args) != 1 {
		fmt.Println(strings.TrimSpace(help))
		return
	}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fatalf("%v", err)
	}

	switch strings.ToLower(args[0]) {
	case "blockheader":
		b := make([]byte, len(data)/2)
		_, err := hex.Decode(b, data)
		if err != nil {
			fatalf("err decoding hex: %s", err)
		}

		var bh legacy.BlockHeader
		err = bh.Scan(b)
		if err != nil {
			fatalf("error decoding: %s", err)
		}

		// The struct doesn't have the hash, so calculate it and print it
		// before pretty printing the header.
		fmt.Printf("Block Hash: %x\n", bh.Hash().Bytes())
		prettyPrint(bh)
	case "block":
		b := make([]byte, len(data)/2)
		_, err := hex.Decode(b, data)
		if err != nil {
			fatalf("err decoding hex: %s", err)
		}

		var block legacy.Block
		err = block.Scan(b)
		if err != nil {
			fatalf("error decoding: %s", err)
		}

		// The struct doesn't have the hash, so calculate it and print it
		// before pretty printing the block
		fmt.Printf("Block Hash: %x\n", block.Hash().Bytes())
		prettyPrint(block)
	case "script":
		b := make([]byte, len(data)/2)
		_, err := hex.Decode(b, data)
		if err != nil {
			fatalf("err decoding hex: %s", err)
		}

		s, err := vm.Disassemble(b)
		if err != nil {
			fatalf("error decoding script: %s", err)
		}
		fmt.Println(s)
	case "txvm":
		b := make([]byte, len(data)/2)
		_, err := hex.Decode(b, data)
		if err != nil {
			fatalf("err decoding hex: %s", err)
		}

		s := txvm.Disassemble(b)
		fmt.Println(s)
	case "tx":
		var tx legacy.Tx
		err := tx.UnmarshalText(data)
		if err != nil {
			fatalf("error decoding: %s", err)
		}
		prettyPrint(tx)
	default:
		fatalf("unrecognized entity `%s`", args[0])
	}
}
