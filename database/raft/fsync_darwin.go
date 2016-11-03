// +build darwin

package raft

import (
	"os"
	"syscall"
)

func fsync(f *os.File) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_FULLFSYNC, 0)
	if errno == 0 {
		return nil
	}
	return errno
}
