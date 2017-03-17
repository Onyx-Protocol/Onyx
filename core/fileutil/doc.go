// Package fileutil contains OS-compatible utilities for writing Chain Core's
// application data.
package fileutil

// DefaultDir returns the directory used to store Chain Core's application data.
func DefaultDir() string {
	return defaultDir()
}
