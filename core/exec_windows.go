package core

import "os"

// These values are used as the process exit status
// to indicate to the process monitor that the next
// invocation of cored should reset the database.
const (
	WinCodeResetBlockchain = 101
	WinCodeResetEverything = 102
)

var WinResetCodeToEnv = map[uint32]string{
	WinCodeResetBlockchain: "blockchain",
	WinCodeResetEverything: "everything",
}

// execSelf assumes the current process is a child of another cored process,
// which is serving as a monitor.
// We simply exit forcefully, and the monitor will restart this program.
// dataToReset should be "blockchain" or "everything" or "".
// (Any other value is treated like "").
func execSelf(dataToReset string) {
	switch dataToReset {
	case "blockchain":
		os.Exit(WinCodeResetBlockchain)
	case "everything":
		os.Exit(WinCodeResetEverything)
	}
	os.Exit(0)
}
