// This file contains tests for the log parity checker.

package testdata

import "chain/log"

// PrintkvTests never executes, but it serves as a simple test for the program.
// Test with (cd ..; go test).
func PrintkvTests() {
	log.Printkv(nil, "k", "v")            // ok
	log.Printkv(nil)                      // zero is ok too
	log.Printkv(nil, []interface{}{0}...) // any 'arg...' is ok too
	log.Printkv(nil, "k")                 // ERROR "odd number of arguments in call to log.Printkv"
	log.Printkv(nil, "k", "v", "k2")      // ERROR "odd number of arguments in call to log.Printkv"

	var log writer
	log.Printkv(nil, "k", "v")       // ok
	log.Printkv(nil)                 // zero is ok too
	log.Printkv(nil, "k")            // ok, log is not the package
	log.Printkv(nil, "k", "v", "k2") // ok, log is not the package
}

type writer struct{}

func (w writer) Printkv(interface{}, ...interface{}) {}
