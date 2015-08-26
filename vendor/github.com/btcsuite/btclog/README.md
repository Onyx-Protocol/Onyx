btclog
======

[![Build Status](https://travis-ci.org/btcsuite/btclog.png?branch=master)]
(https://travis-ci.org/btcsuite/btclog)

Package btclog implements a subsystem aware logger backed by seelog.

Seelog allows you to specify different levels per backend such as console and
file, but it doesn't support levels per subsystem well.  You can create multiple
loggers, but when those are backed by a file, they have to go to different
files.  That is where this package comes in.  It provides a SubsystemLogger
which accepts the backend seelog logger to do the real work.  Each instance of a
SubsystemLogger then allows you specify (and retrieve) an individual level per
subsystem.  All messages are then passed along to the backend seelog logger.

## Documentation

Full `go doc` style documentation for the project can be viewed online without
installing this package by using the GoDoc site here:
http://godoc.org/github.com/btcsuite/btclog

You can also view the documentation locally once the package is installed with
the `godoc` tool by running `godoc -http=":6060"` and pointing your browser to
http://localhost:6060/pkg/github.com/btcsuite/btclog

## Installation

```bash
$ go get github.com/btcsuite/btclog
```

## GPG Verification Key

All official release tags are signed by Conformal so users can ensure the code
has not been tampered with and is coming from the btcsuite developers.  To
verify the signature perform the following:

- Download the public key from the Conformal website at
  https://opensource.conformal.com/GIT-GPG-KEY-conformal.txt

- Import the public key into your GPG keyring:
  ```bash
  gpg --import GIT-GPG-KEY-conformal.txt
  ```

- Verify the release tag with the following command where `TAG_NAME` is a
  placeholder for the specific tag:
  ```bash
  git tag -v TAG_NAME
  ```

## License

Package btclog is licensed under the liberal ISC License.
