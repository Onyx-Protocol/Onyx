package core

import (
	"os"
	"path/filepath"
)

// DataDirFromEnvironment returns the directory to use
// for reading config and storing variable data.
// It returns $CHAIN_CORE_DATA,
// or, if that is empty, $HOME/.chaincore.
func DataDirFromEnvironment() string {
	if s := os.Getenv("CHAIN_CORE_DATA"); s != "" {
		return s
	}
	return filepath.Join(os.Getenv("HOME"), ".chaincore")
}
