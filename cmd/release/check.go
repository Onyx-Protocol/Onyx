package main

import (
	"bytes"

	"chain/build/release"
)

func check() {
	err := release.Check()
	if err != nil {
		fatalf("%s\n", err)
	}
	// TODO(kr): ensure tagged releases match the tags
	// TODO(kr): ensure commit hashes exist
	// TODO(kr): ensure that commit(vprev) is ancestor of commit(vcur) for all vprev < vcur.
	fatalf("TODO(kr): ensure tagged releases match the tags\n")
}

// checkRelease checks that the inputs are consistent with
// each other and with the files in $CHAIN and ${CHAIN}prv.
// If it finds a problem, it prints an error message and
// exits with a nonzero status.
func checkRelease(p product, d *release.Definition) {
	branch := release.Branch(d.Version)

	_, err := chain.git("fetch")
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

	if p.prv {
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
}
