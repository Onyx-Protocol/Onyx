package fileutil

import (
	"os"
	"path/filepath"
)

func defaultDir() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Chain Core")
}
