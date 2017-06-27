// +build darwin

// On Darwin, fsync() does not guarantee data integrity if the system loses power
// or crashes. The recommended solution is to use the F_FULLSYNC fcntl.
//
// For more, see the Apple man page for fsync:
// https://developer.apple.com/legacy/library/documentation/Darwin/Reference/ManPages/man2/fsync.2.html

package sinkdb

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
