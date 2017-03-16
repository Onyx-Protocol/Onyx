//+build linux darwin

package core

import (
	"os"
	"strings"
	"syscall"
)

// execSelf execs the currently-running binary with os.Args.
// If dataToReset is nonempty, it adds RESET=dataToReset
// to the environment of the new process.
// dataToReset should be "blockchain" or "everything" or "".
// (Any other value is treated like "").
func execSelf(dataToReset string) {
	binpath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	env := os.Environ()
	if dataToReset != "" {
		env = mergeEnvLists([]string{"RESET=" + dataToReset}, env)
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
