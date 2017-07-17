package main

import (
	"bytes"
	"os"
	"os/exec"

	"chain/build/release"
)

// TODO(kr): we keep writing VCS operations like this.
// Should we factor it out into a package chain/vcs/git?

var (
	chain    = &repo{dir: os.Getenv("CHAIN")}
	chainprv = &repo{dir: os.Getenv("CHAIN") + "prv"}
)

type repo struct {
	dir string
}

func (r *repo) git(arg ...string) ([]byte, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = r.dir
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func (r *repo) head() (string, error) {
	b, err := chain.git("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(b)), nil
}

func tag(p product, d *release.Definition) string {
	name := p.name + "-" + d.Version
	_, err := chain.git("tag", "-m", name, name, d.ChainCommit)
	if err != nil {
		fatalf("error: %s\n", err)
	}
	if p.prv {
		_, err := chainprv.git("tag", "-m", name, name, d.ChainprvCommit)
		if err != nil {
			untag(p, d, name)
			fatalf("error: %s\n", err)
		}
	}
	return name
}

func untag(p product, d *release.Definition, name string) {
	chain.git("tag", "-d", name)
	if p.prv {
		chainprv.git("tag", "-d", name)
	}
}
