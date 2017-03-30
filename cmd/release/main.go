package main

import (
	"fmt"
	"os"
)

var (
	chain    = &repo{dir: os.Getenv("CHAIN")}
	chainprv = &repo{dir: os.Getenv("CHAIN") + "prv"}
)

type product struct {
	name string // e.g. chain-core-server, sdk-java, chain-enclave
	prv  bool   // whether it's built from chainprv

	// Build builds and packages the release, leaving the
	// results in one or more files on disk.
	// It returns a slice of file names for whatever it built.
	// e.g. chain-core-server-1.1-linux-amd64.tar.gz
	build func(p product, version, tagName string) ([]string, error)
}

var products = []product{
	chainCoreServer,
}

// Exit status 2 means usage error.
// Exit status 1 means something else went wrong.

type config struct {
	product string
	version string
	branch  string
	pubid   string
	prvid   string
	doPrv   bool
}

func main() {
	if len(os.Args) != 5 && len(os.Args) != 6 {
		usage()
	}
	c := &config{
		product: os.Args[1],
		version: os.Args[2],
		branch:  os.Args[3],
		pubid:   os.Args[4],
	}
	if len(os.Args) == 6 {
		c.doPrv = true
		c.prvid = os.Args[5]
		detectRepoCommits(c) // swap pub and prv if necessary
	}

	fmt.Println("release", c)

	for _, p := range products {
		if p.name == c.product {
			release(p, c)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "unknown product %s\n", c.product)
	os.Exit(1)
}

func release(p product, c *config) {
	validate(p, c)
	tagName := tag(p, c)
	files, err := p.build(p, c.version, tagName)
	if err != nil {
		untag(c, tagName)
		fatalf("error: %s\n", err)
	}
	upload(files)
}

// Validate checks that the inputs are consistent with
// each other and with the files in $CHAIN and ${CHAIN}prv.
// If it finds a problem, it prints an error message and
// exits with a nonzero status.
func validate(p product, c *config) {
	// tktk write this
}

func tag(p product, c *config) string {
	name := p.name + "-" + c.version
	_, err := chain.git("tag", name, c.pubid)
	if err != nil {
		fatalf("error: %s\n", err)
	}
	if c.doPrv {
		_, err := chainprv.git("tag", name, c.prvid)
		if err != nil {
			untag(c, name)
			fatalf("error: %s\n", err)
		}
	}
	return name
}

func untag(c *config, name string) {
	chain.git("tag", "-d", name)
	if c.doPrv {
		chain.git("tag", "-d", name)
	}
}

func upload(files []string) {
}

func detectRepoCommits(c *config) {
	if chain.hasCommit(c.prvid) && chainprv.hasCommit(c.pubid) {
		c.pubid, c.prvid = c.prvid, c.pubid
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: release [name] [vers] [branch] [commit] [prv-commit]\n")
	// tktk write more help text
	os.Exit(2)
}
