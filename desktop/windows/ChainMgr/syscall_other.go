//+build !windows

package main

import (
	"errors"
	"runtime"
)

func dropAdminPrivs() error {
	return errors.New("not implemented on " + runtime.GOOS)
}
