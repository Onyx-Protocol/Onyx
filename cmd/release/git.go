package main

import (
	"os"
	"os/exec"
)

// TODO(kr): we keep writing VCS operations like this.
// Should we factor it out into a package chain/vcs/git?

type repo struct {
	dir string
}

func (r *repo) git(arg ...string) ([]byte, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = r.dir
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
