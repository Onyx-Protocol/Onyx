// This file contains tests for the log parity checker.

package testdata

import "chain/log"

// This function never executes, but it serves as a simple test for the program.
// Test with (cd ..; go test).
func WriteTests() {
	log.Write(nil, "k", "v")       // ok
	log.Write(nil)                 // zero is ok too
	log.Write(nil, []int{0}...)    // any 'arg...' is ok too
	log.Write(nil, "k")            // ERROR "odd number of arguments in call to log.Write"
	log.Write(nil, "k", "v", "k2") // ERROR "odd number of arguments in call to log.Write"

	var log writer
	log.Write(nil, "k", "v")       // ok
	log.Write(nil)                 // zero is ok too
	log.Write(nil, "k")            // ok, log is not the package
	log.Write(nil, "k", "v", "k2") // ok, log is not the package
}

type writer struct{}

func (w writer) Write(interface{}, ...interface{}) {}
