// Command txvalidate validates a Chain Protocol transaction.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"chain/protocol/txvm"
)

const help = `Usage: txvalidate [-t] <tx

Command txvalidate reads a transaction from stdin,
executes it, and, if valid, prints its txid to stdout.

On Mac OS X, to validate a tx from the pasteboard:

	pbpaste|txvalidate

Exit code 0 indicates success.
Exit code 1 indicates an invalid transaction.
Exit code 2 indicates a usage or I/O error.

Flags:
`

var (
	flagT = flag.Bool("t", false, "print execution trace to stderr")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, help)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 0 {
		fmt.Fprint(os.Stderr, help)
		flag.PrintDefaults()
		os.Exit(2)
	}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	opts := []txvm.Option{
		txvm.TraceError(func(e error) { err = e }),
	}
	if *flagT {
		opts = append(opts, txvm.TraceOp(trace))
	}

	txid, ok := txvm.Validate(data, opts...)
	if !ok {
		fmt.Fprintln(os.Stderr, "invalid tx:", err)
		os.Exit(1)
	}
	fmt.Print(txid)
}

func trace(_ byte, data []byte, vm txvm.VM) {
	stack := vm.Stack(txvm.StackData)
	for i := 0; i < stack.Len(); i++ {
		// TODO(kr): format items better
		// (prob "encode" the item, then disassemble the encoded bytes).
		fmt.Fprintf(os.Stderr, "%x", stack.Element(i))
		fmt.Fprint(os.Stderr, " ")
	}
	fmt.Fprintln(os.Stderr, ".", txvm.Disassemble(data[vm.PC():]))
	// TOOD(kr): print changes to other stacks (or full stack contents?)
}
