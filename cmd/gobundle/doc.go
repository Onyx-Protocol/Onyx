/*

Command gobundle encodes filesystem files as Go source.

Usage:

	gobundle [-package name] [-symbol name] [src]

If src is a directory, it reads all the plain files in src,
recursively, and prints their contents to
stdout as Go source. The generated file will define a single
variable at package scope.

	var Files = map[string]string{...}

The keys are file paths relative to src,
and the values are the file contents.

It skips all special files.

If src is a regular file, it makes a single entry in Files
using the base name of src as the key.

Flag -package controls the generated package name.
(The default is "main".)

Flag -symbol controls the generated variable name
of the map. (The default is "Files".)

*/
package main
