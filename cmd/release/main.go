package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	stableBranch, err := regexp.MatchString("^\\d+\\.\\d+-stable$", c.branch)
	if err != nil {
		fatalf("error: %s\n", err)
	}
	if c.branch != "main" && !stableBranch {
		fatalf("error: invalid branch %s\n", c.branch)
	}

	_, err = chain.git("fetch")
	if err != nil {
		fatalf("error: %s\n", err)
	}

	_, err = chainprv.git("fetch")
	if err != nil {
		fatalf("error: %s\n", err)
	}

	_, err = chain.git("checkout", c.branch)
	if err != nil {
		fatalf("error: %s\n", err)
	}

	commitBytes, err := chain.git("rev-parse", "HEAD")
	if err != nil {
		fatalf("error: %s\n", err)
	}
	commit := string(bytes.TrimSpace(commitBytes))
	if commit != c.pubid {
		fatalf("error: got commit %s expected %s on chain\n", commit, c.pubid)
	}

	if c.doPrv {
		_, err = chainprv.git("checkout", c.branch)
		if err != nil {
			fatalf("error: %s\n", err)
		}

		commitBytes, err := chain.git("rev-parse", "HEAD")
		if err != nil {
			fatalf("error: %s\n", err)
		}
		commit := string(bytes.TrimSpace(commitBytes))
		if commit != c.pubid {
			fatalf("error: got commit %s expected %s on chainprv\n", commit, c.pubid)
		}
	}

	var versionPrefix string
	if stableBranch {
		versionPrefix = strings.Split(c.branch, "-")[0]
	}

	if c.doPrv {
		checkTag(chainprv, c, versionPrefix)
	} else {
		checkTag(chain, c, versionPrefix)
	}
}

func checkTag(r *repo, c *config, versionPrefix string) {
	proposedVersion := stringToVersion(c.version)
	if proposedVersion[2] == 0 && c.branch != "main" {
		fatalf("error: %s can only be released from main\n", c.version)
	} else if proposedVersion[2] != 0 && c.branch == "main" {
		fatalf("error: %s can only be released from a stable branch\n", c.version)
	}

	search := c.product + "-"
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
		tag = bytes.TrimPrefix(tag, []byte(c.product+"-"))
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

	fatalf("error: %s is not the current or next version and is not releasable\n", c.version)
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
