package main

import (
	"flag"
	"fmt"
	"os"

	"chain/build/release"
)

const help = `Command release builds and publishes Chain software.

Usage:

    release [-t] [product] [version]
    release -less [version a] [version b]
    release -checktags

See 'go doc chain/cmd/release' for detailed documentation.

When run with no flags, command release finds a release
definition, builds the release, including obtaining any
necessary signatures, and publishes it. If version is
omitted, it uses the latest release for the given product.

Flag -t runs in test mode: it does not read release.txt and
it does not publish the built artifacts; instead, it builds
the product as if it were being released and leaves the
built artifacts in the local filesystem for further testing.
It builds the current HEAD ref of the git repository in
$CHAIN and ${CHAIN}prv.

Flag -less compares two version strings for inequality.

Flag -checktags skips the whole build and publish process,
instead checking that git tags match the commit hashes
listed in release.txt.
`

var (
	test      = flag.Bool("t", false, "test the release process")
	less      = flag.Bool("less", false, "compare two version strings")
	checktags = flag.Bool("checktags", false, "check validity of release tags, do not build")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if *checktags {
		check()
		return
	} else if *less {
		doLess()
		return
	}

	// build mode (not checktags or version comparison)
	if n := flag.NArg(); n < 1 || n > 2 {
		usage()
	}

	definition := release.Get
	if *test {
		definition = temporaryDefinition
	}
	def, err := definition(flag.Arg(0), flag.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("release", *def)

	for _, p := range products {
		if p.name == def.Product {
			doRelease(p, def)
			return
		}
	}
	fmt.Fprintln(os.Stderr, "unknown product", def.Product)
	os.Exit(1)
}

func doRelease(p product, def *release.Definition) {
	checkRelease(p, def)
	tagName := tag(p, def)
	files, err := p.build(p, def.Version, tagName)
	if err != nil {
		untag(p, def, tagName)
		fatalf("error: %s\n", err)
	}
	upload(files)
}

func temporaryDefinition(product, version string) (*release.Definition, error) {
	def := &release.Definition{
		Product:        product,
		Version:        version, // note: may not be a valid version string, that is ok!
		ChainCommit:    "aaaa",
		ChainprvCommit: "bbbb",
		Codename:       "Code Name", // TODO(kr): put something useful in here
	}
	return def, nil
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, help)
	os.Exit(2)
}
