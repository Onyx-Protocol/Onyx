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
)

func main() {
	log.SetPrefix("gobundle: ")
	log.SetFlags(log.Lshortfile)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gobundle [-package name] [dir]")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	dir := flag.Arg(0)

	fmt.Println("package", *pkg)
	fmt.Println("var Files = map[string]string{")

	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalln(err)
	}
	for _, info := range infos {
		if info.Mode()&os.ModeType != 0 {
			continue
		}
		b, err := ioutil.ReadFile(filepath.Join(dir, info.Name()))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%q: %q,\n", info.Name(), b)
	}
	fmt.Println("}")
}
