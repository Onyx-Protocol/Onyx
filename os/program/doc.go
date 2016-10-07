// Package program gets the filepath to the current program.
package program

// Path returns the filepath to the program executing in the current process.
func Path() (string, error) {
	return path()
}
