//+build linux darwin

package core

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// execSelf execs the currently-running binary with os.Args.
// If dataToReset is nonempty, it adds RESET=dataToReset
// to the environment of the new process.
// dataToReset should be "blockchain" or "everything" or "".
// (Any other value is treated like "").
func execSelf(dataToReset string) {
	// TODO(kr): use a more reliable way of finding the current program
	// e.g. 'readlink /proc/self/exe' on linux, or the "apple" argument to main.
	// See https://unixjunkie.blogspot.com/2006/02/char-apple-argument-vector.html.
	binpath, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(err)
	}

	var env []string
	if dataToReset != "" {
		env = mergeEnvLists([]string{"RESET=" + dataToReset}, os.Environ())
	}
	err = syscall.Exec(binpath, os.Args, env)
	if err != nil {
		panic(err)
	}
}

// mergeEnvLists merges the two environment lists such that
// variables with the same name in "in" replace those in "out".
// This always returns a newly allocated slice.
func mergeEnvLists(in, out []string) []string {
	out = append([]string(nil), out...)
NextVar:
	for _, inkv := range in {
		k := strings.SplitAfterN(inkv, "=", 2)[0]
		for i, outkv := range out {
			if strings.HasPrefix(outkv, k) {
				out[i] = inkv
				continue NextVar
			}
		}
		out = append(out, inkv)
	}
	return out
}
