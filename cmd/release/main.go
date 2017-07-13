package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"chain/build/release"
)

var (
	flagT     = flag.Bool("t", false, "test the release process")
	flagCheck = flag.Bool("check", false, "check validity of release tags, do not build")
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

func main() {
	flag.Parse()

	if n := flag.NArg(); n != 2 && n != 1 {
		usage()
	}

	product := flag.Arg(1)
	version := flag.Arg(2)
	def, err := release.Get(product, version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
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
	validate(p, def)
	tagName := tag(p, def)
	files, err := p.build(p, def.Version, tagName)
	if err != nil {
		untag(def, tagName)
		fatalf("error: %s\n", err)
	}
	upload(files)
}

// Validate checks that the inputs are consistent with
// each other and with the files in $CHAIN and ${CHAIN}prv.
// If it finds a problem, it prints an error message and
// exits with a nonzero status.
func validate(p product, d *release.Definition) {
	branch := release.Branch(d.Version)

	stableBranch, err := regexp.MatchString("^\\d+\\.\\d+-stable$", branch)
	if err != nil {
		fatalf("error: %s\n", err)
	}
	if branch != "main" && !stableBranch {
		fatalf("error: invalid branch %s\n", branch)
	}

	_, err = chain.git("fetch")
	if err != nil {
		fatalf("error: %s\n", err)
	}

	_, err = chainprv.git("fetch")
	if err != nil {
		fatalf("error: %s\n", err)
	}

	_, err = chain.git("checkout", branch)
	if err != nil {
		fatalf("error: %s\n", err)
	}

	commitBytes, err := chain.git("rev-parse", "HEAD")
	if err != nil {
		fatalf("error: %s\n", err)
	}
	commit := string(bytes.TrimSpace(commitBytes))
	if commit != d.ChainCommit {
		fatalf("error: got commit %s expected %s on chain\n", commit, d.ChainCommit)
	}

	if doPrv(d) {
		_, err = chainprv.git("checkout", branch)
		if err != nil {
			fatalf("error: %s\n", err)
		}

		commitBytes, err := chain.git("rev-parse", "HEAD")
		if err != nil {
			fatalf("error: %s\n", err)
		}
		commit := string(bytes.TrimSpace(commitBytes))
		if commit != d.ChainCommit {
			fatalf("error: got commit %s expected %s on chainprv\n", commit, d.ChainCommit)
		}
	}

	var versionPrefix string
	if stableBranch {
		versionPrefix = strings.Split(branch, "-")[0]
	}

	if doPrv(d) {
		checkTag(chainprv, d, versionPrefix)
	} else {
		checkTag(chain, d, versionPrefix)
	}
}

func doPrv(d *release.Definition) bool {
	return d.ChainprvCommit != "na"
}

func checkTag(r *repo, d *release.Definition, versionPrefix string) {
	branch := release.Branch(d.Version)
	proposedVersion := stringToVersion(d.Version)
	if proposedVersion[2] == 0 && branch != "main" {
		fatalf("error: %s can only be released from main\n", d.Version)
	} else if proposedVersion[2] != 0 && branch == "main" {
		fatalf("error: %s can only be released from a stable branch\n", d.Version)
	}

	search := d.Product + "-"
	if versionPrefix != "" {
		search += versionPrefix + "*"
	} else {
		search += "*.*.0"
	}
	tagBytes, err := r.git("tag", "-l", search)
	if err != nil {
		fatalf("error: %s\n", err)
	}

	tags := bytes.Split(tagBytes, []byte("\n"))
	var checkVersion [3]int
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}
		tag = bytes.TrimPrefix(tag, []byte(d.Product+"-"))
		tagVersion := stringToVersion(string(tag))
		if cmpVersions(tagVersion, checkVersion) == 1 {
			checkVersion = tagVersion
		}
	}

	if cmpVersions(proposedVersion, checkVersion) == 0 {
		return
	}

	if versionPrefix != "" {
		checkVersion[2]++
		if cmpVersions(proposedVersion, checkVersion) == 0 {
			return
		}
	} else {
		check2 := checkVersion
		check3 := checkVersion

		check2[1]++
		check3[0]++
		check3[1] = 0

		if cmpVersions(proposedVersion, check2) == 0 || cmpVersions(proposedVersion, check3) == 0 {
			return
		}
	}

	fatalf("error: %s is not the current or next version and is not releasable\n", d.Version)
}

func stringToVersion(str string) [3]int {
	versionParts := strings.Split(str, ".")
	var tagVersion [3]int
	for i, versionPart := range versionParts {
		versionNum, err := strconv.Atoi(versionPart)
		if err != nil {
			fatalf("error: %s\n", err)
		}
		tagVersion[i] = versionNum
	}
	return tagVersion
}

func cmpVersions(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] > b[i] {
			return 1
		} else if a[i] < b[i] {
			return -1
		}
	}
	return 0
}

func tag(p product, d *release.Definition) string {
	name := p.name + "-" + d.Version
	_, err := chain.git("tag", name, d.ChainCommit)
	if err != nil {
		fatalf("error: %s\n", err)
	}
	if doPrv(d) {
		_, err := chainprv.git("tag", name, d.ChainprvCommit)
		if err != nil {
			untag(d, name)
			fatalf("error: %s\n", err)
		}
	}
	return name
}

func untag(d *release.Definition, name string) {
	chain.git("tag", "-d", name)
	if doPrv(d) {
		chain.git("tag", "-d", name)
	}
}

func upload(files []string) {
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
