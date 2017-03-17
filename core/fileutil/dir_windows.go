package fileutil

import (
	"os"
	"path/filepath"
)

func defaultDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), `Chain Core`)
}
