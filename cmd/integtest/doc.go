/*

Command integtest compiles and runs a command
to perform integration tests on Chain Core
and related systems.
It sets up common resources used by most or all integration tests,
so the tests themselves don't have to.

Its first argument is the import path of a command package
to compile and execute.
Subsequent arguments are passed on to the test process.

Flags

Flag '-t [duration]' aborts the test after the given duration.
The default is 15 minutes.

Files and Environment

This command puts all temporary files in various subdirectories
of $HOME/integration.
This scratch space is cleaned up before each test,
but left intact after the test
to assist in debugging a failed test.
(Note, however, that the test process itself
can write files anywhere.)

It reads GitHub credentials from .netrc.

It requires Postgres and Go installed.

Test Environment

This tool always sets some values for the test process:

  CHAIN          a clean checkout of the chain repo (and ${CHAIN}prv)
  DB_URL_TEST    the URL of a new, empty postgres cluster
  GOPATH         the Go workspace containing $CHAIN
  (working dir)  an empty scratch directory

*/
package main
