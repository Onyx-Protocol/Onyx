//+build !windows,!darwin

package fileutil

import (
	"os"
	"path/filepath"
)

func DefaultDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cored")
}
