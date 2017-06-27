// +build !darwin

package sinkdb

import "os"

func fsync(f *os.File) error {
	return f.Sync()
}
