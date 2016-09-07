/*

Command gobundle encodes filesystem files as Go source.

Usage:

	gobundle [-package name] [path]

If path is a directory, it reads all the plain files in path,
and prints their contents to
stdout as Go source. The generated file will define a single
variable "Files" at package scope.

	var Files = map[string]string{...}

The keys are the file names in path, and the values are the file
contents.

It does not run recursively, and skips all special files.

If path is a regular file, it makes a single entry in Files
using the base name of path as the key.

*/
package main
