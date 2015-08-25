// +build ignore

// usage: go run gen.go [package] [symbol] [file]
// ex: go run gen.go wallets keySQL keys.sql
// creates keys.sql.go
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	pkg := os.Args[1]
	symbol := os.Args[2]
	file := os.Args[3]
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Create(file + ".go")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(f, "package %s\n\n", pkg)
	fmt.Fprintf(f, "//go:generate go run gen.go %s %s %s\n", pkg, symbol, file)
	if bytes.IndexByte(b, '`') > -1 {
		fmt.Fprintf(f, "const %s = %q\n", symbol, b)
	} else {
		fmt.Fprintf(f, "const %s = `%s`\n", symbol, b)
	}
	f.Close()
}
