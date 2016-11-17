package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	pkg = flag.String("package", "main", "`name` of generated package")
	sym = flag.String("symbol", "Files", "`name` of map variable")
)

func main() {
	log.SetPrefix("gobundle: ")
	log.SetFlags(log.Lshortfile)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gobundle [-package name] [-symbol name] [src]")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	root := flag.Arg(0)

	fmt.Println("package", *pkg)
	fmt.Println("var", *sym, "= map[string]string{")

	filepath.Walk(root, func(path string, ent os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)
			return nil
		} else if ent.Mode()&os.ModeType != 0 {
			return nil
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			log.Println(err)
			return nil
		}
		rel := ent.Name() // in case root is a file
		if path != root {
			rel, _ = filepath.Rel(root, path) // #nosec
		}
		fmt.Printf("%q: %q,\n", rel, b)
		return nil
	})

	fmt.Println("}")
}
