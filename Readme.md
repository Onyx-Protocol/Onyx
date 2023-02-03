

## Onyx Core 

**Onyx Core** is software designed to operate and connect to highly scalable permissioned blockchain networks conforming to the Onyx Protocol. Each network maintains a cryptographically-secured transaction log, known as a blockchain, which allows participants to define, issue, and transfer digital assets on a multi-asset shared ledger. Digital assets share a common, interoperable format and can represent any units of value that are guaranteed by a trusted issuer — such as currencies, bonds, securities, IOUs, or loyalty points. Each Onyx Core holds a copy of the ledger and independently validates each update, or “block,” while a federation of block signers ensures global consistency of the ledger.

**Onyx Core Developer Edition** is a free, downloadable version of Onyx Core that is open source and licensed under the AGPL. Individuals and organizations use Onyx Core Developer Edition to learn, experiment, and build prototypes.

Onyx Core Developer Edition can be run locally on Mac, Windows, or Linux to create a new blockchain network, connect to an existing blockchain network, or connect to the public Onyx testnet, operated by Onyx, Microsoft, and Cornell University’s IC3.

For more information about how to use Onyx Core Developer Edition, see the docs: https://Onyx.org/docs

## Download

To install Onyx Core Developer Edition on Mac, Windows, or Linux, please visit [our downloads page](https://Onyx.org/docs/core/get-started/install).

## Contributing

Onyx has adopted the code of conduct defined by the Contributor Covenant. It can be read in full [here](https://github.com/Onyx-Protocol/Onyx/blob/main/CODE_OF_CONDUCT.md).
This repository is the canonical source for Onyx Core Developer Edition. Consequently, Onyx engineers actively maintain this repository.
If you are interested in contributing to this code base, please read our [issue](https://github.com/Onyx-Protocol/Onyx/blob/main/.github/ISSUE_TEMPLATE.md) and [pull request](https://github.com/Onyx-Protocol/Onyx/blob/main/.github/PULL_REQUEST_TEMPLATE.md) templates first.

## Building from source

* [Go](https://golang.org/doc/install) version 1.8, with $GOPATH set to your
  preferred directory
* Postgres (we suggest [Postgres.app](http://postgresapp.com/)),
  along with the [command line tools](http://postgresapp.com/documentation/cli-tools.html)
* [protoc](https://github.com/google/protobuf#protocol-compiler-installation) 3.1.0 and
  [protoc-gen-g](https://github.com/golang/protobuf/protoc-gen-go), if you need to compile protos
* [CMake](https://cmake.org/) 3.4 or later, to compile RocksDB and its dependencies

### Environment

Set the `ONYX` environment variable, in `.profile` in your home
directory, to point to the root of the Onyx source code repo:

```sh
export ONYX=$(go env GOPATH)/src/Onyx
```

You should also add `$ONYX/bin` to your path (as well as
`$(go env GOPATH)/bin`, if it isn’t already):

```sh
PATH=$(go env GOPATH)/bin:$ONYX/bin:$PATH
```

You might want to open a new terminal window to pick up the change.

### Installation

Clone this repository to `$ONYX`:

```sh
$ git clone https://github.com/Onyx-Protocol/Onyx $ONYX
$ cd $ONYX
```

You can build Onyx Core using the `build-cored-release` script.
The build product allows connections over HTTP, unauthenticated
requests from localhost, and the ability to reset the Onyx Core.

`build-cored-release` accepts a accepts a Git ref (branch, tag, or commit SHA)
from the Onyx repository and an output directory:

```sh
$ ./bin/build-cored-release Onyx-core-server-1.2.0 .
```

This will create two binaries in the current directory:

* [cored](https://Onyx.org/docs/core/reference/cored): the Onyx Core daemon and API server
* [corectl](https://Onyx.org/docs/core/reference/corectl): control functions for a Onyx Core

Set up the database:

```sh
$ createdb core
```

Start Onyx Core:

```sh
$ ./cored
```

Access the dashboard:

```sh
$ open http://localhost:1999/
```

Run tests:

```sh
$ go test $(go list ./... | grep -v vendor)
```

### Building from source

There are four build tags that change the behavior of the resulting binary:

* `reset`: allows the core database to be reset through the api
* `localhost_auth`: allows unauthenticated requests on the loopback device (localhost)
* `no_mockhsm`: disables the MockHSM provided for development
* `http_ok`: allows plain HTTP requests
* `init_cluster`: automatically creates a single process cluster

The default build process creates a binary with three build tags enabled for a
friendlier experience. To build from source with build tags, use the following
command:

> NOTE: when building from source, make sure to check out a specific
> tag to build. The `main` branch is **not considered** stable, and may
> contain in progress features or an inconsistent experience.

```sh
$ go build -tags 'http_ok localhost_auth init_cluster' Onyx/cmd/cored
$ go build Onyx/cmd/corectl
```

## Developing Onyx Core

### Updating the schema with migrations

```sh
$ go run cmd/dumpschema/main.go
```

### Dependencies

To add or update a Go dependency at import path `x`, do the following:

Copy the code from the package's directory
to `$ONYX/vendor/x`. For example, to vendor the package
`github.com/kr/pretty`, run

```sh
$ mkdir -p $ONYX/vendor/github.com/kr
$ rm -r $ONYX/vendor/github.com/kr/pretty
$ cp -r $(go list -f {{.Dir}} github.com/kr/pretty) $ONYX/vendor/github.com/kr/pretty
$ rm -rf $ONYX/vendor/github.com/kr/pretty/.git
```

(Note: don’t put a trailing slash (`/`) on these paths.
It can change the behavior of cp and put the files
in the wrong place.)

In your commit message, include the commit hash of the upstream repo
for the dependency. (You can find this with `git rev-parse HEAD` in
the upstream repo.) Also, make sure the upstream working tree is clean.
(Check with `git status`.)

## License

Onyx Core Developer Edition is licensed under the terms of the [GNU
Affero General Public License Version 3 (AGPL)](LICENSE).

The Onyx Java SDK (`/sdk/java`) is licensed under the terms of the
[Apache License Version 2.0](sdk/java/LICENSE).
