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
		fmt.Fprintln(os.Stderr, "Usage: gobundle [-package name] [path]")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	name := flag.Arg(0)

	fmt.Println("package", *pkg)
	fmt.Println("var Files = map[string]string{")

	switch info, err := os.Stat(name); {
	case err != nil:
		log.Fatalln(err)
	case info.Mode().IsDir():
		entries, err := ioutil.ReadDir(name)
		if err != nil {
			log.Fatalln(err)
		}
		for _, ent := range entries {
			if ent.Mode()&os.ModeType != 0 {
				continue
			}
			b, err := ioutil.ReadFile(filepath.Join(name, ent.Name()))
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("%q: %q,\n", ent.Name(), b)
		}
	case info.Mode().IsRegular():
		b, err := ioutil.ReadFile(name)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%q: %q,\n", info.Name(), b)
	default:
		log.Fatalln(name, "unknown file type", info.Mode())
	}

	fmt.Println("}")
}
