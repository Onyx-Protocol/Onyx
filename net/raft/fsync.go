// +build !darwin

package raft

import "os"

func fsync(f *os.File) error {
	return f.Sync()
}
