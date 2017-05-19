// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file at
// https://github.com/golang/go/blob/master/LICENSE.

/*

chainvet examines Go source code and reports suspicious constructs, such as
Printf calls whose arguments do not align with the format string. chainvet
uses heuristics that do not guarantee all reports are genuine problems,
but it can find errors not caught by the compilers.

It can be invoked two ways:

By files:
	chainvet source/directory/*.go
vets the files named, all of which must be in the same package.

By directory:
	chainvet source/directory
recursively descends the directory, vetting each package it finds.

chainvet's exit code is 2 for erroneous invocation of the tool, 1 if a
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

Parity of log.Write and log.Fatal

Flag: -logparity

Calls to chain/log.Write that have an odd number of key/value arguments.


Use of http.DefaultClient

Flag: -defaulthttp

Calls to http.DefaultClient in Chain packages that should be using a
TLS-configured http client.


Other flags

These flags configure the behavior of chainvet:

	-all (default true)
		Check everything; disabled if any explicit check is requested.
	-v
		Verbose mode
*/
package main // import "chain/cmd/chainvet"
