package main

import (
	"flag"
	"fmt"
	"os"

	"chain/build/release"
)

func doLess() {
	if flag.NArg() != 2 {
		usage()
	}
	a := flag.Arg(0)
	b := flag.Arg(1)
	if err := release.CheckVersion(a); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := release.CheckVersion(b); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if release.Less(a, b) {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
