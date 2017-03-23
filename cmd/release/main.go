package main

import (
	"fmt"
	"os"
)

type product struct {
	name string // e.g. chain-core-server, sdk-java, chain-enclave
	prv  bool   // whether it's built from chainprv

	// build builds and packages the release, leaving the
	// results in files on disk. It returns a slice of filepaths
	// for whatever it built.
	// e.g. chain-core-server-1.1-linux-amd64.tar.gz
	build func(p product, version, tagName string) ([]string, error)
}

var products = []product{
	chainCoreServer,
}

// Exit status 2 means usage error.
// Exit status 1 means something else went wrong.

func main() {
	if len(os.Args) != 5 && len(os.Args) != 6 {
		usage()
	}
	name := os.Args[1]
	version := os.Args[2]
	branch := os.Args[3]
	commit := os.Args[4]
	var prvCommit string
	if len(os.Args) == 6 {
		prvCommit = os.Args[5]
	}

	fmt.Println("release", name, version, branch, commit, prvCommit)

	for _, p := range products {
		if p.name == name {
			release(p, version, branch, commit, prvCommit)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "unknown product %s\n", name)
	os.Exit(1)
}

func release(p product, version, branch, commit, prvCommit string) {
	validate(p, version, branch, commit, prvCommit)
	tagName := tag(p, version, branch, commit, prvCommit)
	files, err := p.build(p, version, tagName)
	if err != nil {
		untag()
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	upload(files)
}

// validate checks that the inputs are consistent with
// each other and with the files in $CHAIN and ${CHAIN}prv.
// If it finds a problem, it prints an error message and
// exits with a nonzero status.
func validate(p product, version, branch, commit, prvCommit string) {
	// tktk write this
}

func tag(p product, version, branch, commit, prvCommit string) string {
	return p.name + "-" + version
}

func untag() {
}

func upload(files []string) {
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: release [name] [vers] [branch] [commit] [prv-commit]\n")
	// tktk write more help text
	os.Exit(2)
}
