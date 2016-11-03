package raft

import (
	"io"
	"os"

	"chain/errors"
)

// writeFile is like ioutil.WriteFile, but it writes safely and atomically.
// (It writes data to a temp file (name+".temp"), syncs data to disk,
// closes the temp file, then atomically renames the temp file to name.)
func writeFile(name string, data []byte, perm os.FileMode) error {
	const suffix = ".temp"
	temp := name + suffix
	f, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return errors.Wrap(err)
	}
	defer f.Close()
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		return errors.Wrap(io.ErrShortWrite)
	} else if err != nil {
		return errors.Wrap(err)
	}
	err = fsync(f)
	if err != nil {
		return errors.Wrap(err)
	}
	err = f.Close()
	if err != nil {
		return errors.Wrap(err)
	}
	return errors.Wrap(os.Rename(temp, name))
}
