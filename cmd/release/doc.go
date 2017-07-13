/*

Command release builds and publishes Chain software.

Usage

Usage of release:

    release [-t] [product] [version]
    release -less [version a] [version b]
    release -checktags

The file $CHAIN/build/release.txt contains a list of release
definitions, one per line. A release definition is a
5-tuple: product, version, chain commit, chainprv commit,
and codename. The fields of each tuple are separated by
whitespace. End-of-line comments can appear on any line,
beginning with the hash '#' character. After stripping
comments, empty lines are skipped.

Build Mode

When run with no flags, command release finds a release
definition, builds the release, including obtaining any
necessary signatures, and publishes it. The arguments denote
an entry in release.txt to build. If version is omitted, it
uses the latest release for the given product.

In this mode, exit status 0 means success, exit status 1
means an unexpected error occurred (such as an I/O error or
build failure), and exit status 2 means usage error.

Flags

Flag -t runs in test mode: it does not read release.txt and
it does not publish the built artifacts; instead, it builds
the product as if it were being released and leaves the
built artifacts in the local filesystem for further testing.
Ordinarily, a release definition is read from release.txt;
flag -t instead constructs a temporary release definition as
follows: the product name and version are taken directly
from the command line arguments, the chain commit and
chainprv commit are taken from the HEAD ref of the git
repository in $CHAIN and ${CHAIN}prv, respectively, and the
codename is generated automatically from various other data
(the exact codename is unspecified and may change).

Flag -less compares two version strings for inequality. It
exits 0 if version a is strictly less than version b, and 1
if not. Exit status 2 indicates a usage error, including if
either version string is invalid.

Flag -checktags skips the whole build and publish process.
Instead it reads release.txt, checking that file for errors,
and verifies that any existing git tags for previously
published releases still match the commits listed in the
release definition. This helps prevent accidentally
modifying the release definition after making a release.
This flag is intended to be useful during continuous
integration processes.
`

*/
package main
