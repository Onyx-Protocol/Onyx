package program

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

//sys	GetModuleFileName(hmodule uint32, buf *uint16, buflen uint32) (n uint32, err error) = GetModuleFileNameW
