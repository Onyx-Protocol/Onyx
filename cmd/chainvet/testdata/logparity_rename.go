// This file contains tests for the log parity checker.

package testdata

import chainlog "chain/log"

// PrintkvTestsWithRename never executes, but it serves as a simple test for
// the program. Test with (cd ..; go test).
func PrintkvTestsWithRename() {
	chainlog.Printkv(nil, "k", "v")       // ok
	chainlog.Printkv(nil)                 // zero is ok too
	chainlog.Printkv(nil, "k")            // ERROR "odd number of arguments in call to chainlog.Printkv"
	chainlog.Printkv(nil, "k", "v", "k2") // ERROR "odd number of arguments in call to chainlog.Printkv"
}
