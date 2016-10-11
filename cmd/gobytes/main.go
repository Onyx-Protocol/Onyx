/*

Command gobytes prints its input as a []byte literal.

Examples:

	$ echo foo | gobytes
	[]byte{0x66, 0x6f, 0x6f, 0xa}

	$ echo foo | gobytes -s
	"foo\n"

Flag -s prints a string literal instead of []byte.

*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var flagS = flag.Bool("s", false, "output string literal instead of []byte")

func main() {
	log.SetFlags(0)
	flag.Parse()
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	if *flagS {
		fmt.Printf("%#q\n", b)
	} else {
		fmt.Printf("%#v\n", b)
	}
}
