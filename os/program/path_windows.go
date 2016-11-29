package program

import (
	"os"
	"syscall"

	chainsyscall "chain-stealth/syscall"
)

func path() (name string, err error) {
	n := uint32(64)
	b := make([]uint16, n)
	w, err := chainsyscall.GetModuleFileName(0, &b[0], n)
	if err != nil {
		return "", os.NewSyscallError("GetModuleFileName", err)
	}
	return syscall.UTF16ToString(b[:w]), nil
}
