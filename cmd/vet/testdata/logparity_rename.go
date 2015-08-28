// This file contains tests for the log parity checker.

package testdata

import chainlog "chain/log"

// This function never executes, but it serves as a simple test for the program.
// Test with (cd ..; go test).
func WriteTestsWithRename() {
	chainlog.Write(nil, "k", "v")       // ok
	chainlog.Write(nil)                 // zero is ok too
	chainlog.Write(nil, "k")            // ERROR "odd number of arguments in call to chainlog.Write"
	chainlog.Write(nil, "k", "v", "k2") // ERROR "odd number of arguments in call to chainlog.Write"
}
