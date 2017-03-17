//+build windows

package fileutil

import (
	"os"
	"path/filepath"
)

func DefaultDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Chain")
}
