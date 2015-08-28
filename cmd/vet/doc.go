// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*

Vet examines Go source code and reports suspicious constructs, such as Printf
calls whose arguments do not align with the format string. Vet uses heuristics
that do not guarantee all reports are genuine problems, but it can find errors
not caught by the compilers.

It can be invoked three ways:

By package, from the go tool:
	go vet package/path/name
vets the package whose path is provided.

By files:
	go tool vet source/directory/*.go
vets the files named, all of which must be in the same package.

By directory:
	go tool vet source/directory
recursively descends the directory, vetting each package it finds.

Vet's exit code is 2 for erroneous invocation of the tool, 1 if a
problem was reported, and 0 otherwise. Note that the tool does not
check every possible problem and depends on unreliable heuristics
so it should be used as guidance only, not as a firm indicator of
program correctness.

By default all checks are performed. If any flags are explicitly set
to true, only those tests are run. Conversely, if any flag is
explicitly set to false, only those tests are disabled.
Thus -printf=true runs the printf check, -printf=false runs all checks
except the printf check.

Available checks:

Parity of log.Write

Flag: -logparity

Calls to chain/log.Write that have an odd number of key/value arguments.


Other flags

These flags configure the behavior of vet:

	-all (default true)
		Check everything; disabled if any explicit check is requested.
	-v
		Verbose mode
*/
package main // import "chain/cmd/vet"
