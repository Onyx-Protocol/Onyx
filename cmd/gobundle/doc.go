/*

Command gobundle encodes filesystem files as Go source.

Usage:

	gobundle [-package name] [dir]

It reads all the plain files in dir, and prints their contents to
stdout as Go source. The generated file will define a single
variable "Files" at package scope.

	var Files = map[string]string{...}

The keys are the file names in dir, and the values are the file
contents.

It does not run recursively, and skips all special files.

*/
package main
