# Chain Core Developer Edition

**Chain Core** is software designed to operate and connect to highly scalable permissioned blockchain networks conforming to the Chain Protocol. Each network maintains a cryptographically-secured transaction log, known as a blockchain, which allows partipicants to define, issue, and transfer digital assets on a multi-asset shared ledger. Digital assets share a common, interoperable format and can represent any units of value that are guaranteed by a trusted issuer — such as currencies, bonds, securities, IOUs, or loyalty points. Each Chain Core holds a copy of the ledger and independently validates each update, or “block,” while a federation of block signers ensures global consistency of the ledger. 

**Chain Core Developer Edition** is a free, downloadable version of Chain Core that is open source and licensed under the AGPL. Individuals and organizations use Chain Core Developer Edition to learn, experiment, and build prototypes.

Chain Core Developer Edition can be run locally on Mac, Windows, or Linux to create a new blockchain network, connect to an existing blockchain network, or connect to the public Chain testnet, operated by Chain, Microsoft, and Cornell University’s IC3.

For more information about how to use Chain Core Developer Edition, see the docs: https://chain.com/docs

## Download

To install Chain Core Developer Edition on Mac, Windows, or Linux, please visit [our downloads page](https://chain.com/docs/core/get-started/install).

## Contributing

Chain has adopted the code of conduct defined by the Contributor Covenant. It can be read in full [here](https://github.com/chain/chain/blob/main/CODE_OF_CONDUCT.md).
This repository is the canonical source for Chain Core Developer Edition. Consequently, Chain engineers actively maintain this repository.
If you are interested in contributing to this code base, please read our [issue](https://github.com/chain/chain/blob/main/.github/ISSUE_TEMPLATE.md) and [pull request](https://github.com/chain/chain/blob/main/.github/PULL_REQUEST_TEMPLATE.md) templates first.

## Building from source

* [Go](https://golang.org/doc/install) version 1.7, with $GOPATH set to your
  preferred directory
* Postgres (we suggest [Postgres.app](http://postgresapp.com/)),
  along with the [command line tools](http://postgresapp.com/documentation/cli-tools.html)
* [protoc](https://github.com/google/protobuf#protocol-compiler-installation),
  if you need to compile protos

### Environment

Set the `CHAIN` environment variable, in `.profile` in your home
directory, to point to the root of the Chain source code repo:

```
export CHAIN=$GOPATH/src/chain
```

You should also add `$CHAIN/bin` to your path (as well as
`$GOPATH/bin`, if it isn’t already):

```
PATH=$GOPATH/bin:$CHAIN/bin:$PATH
```

You might want to open a new terminal window to pick up the change.

### Installation

Build and install from source:

```
$ git clone https://github.com/chain/chain $CHAIN
$ cd $CHAIN
$ go install ./cmd/...
```

Set up the database:

```
$ createdb core
```

Start Chain Core:

```
$ cored
```

Access the dashboard:

```
$ open http://localhost:1999/
```

Run tests:

```
$ go test $(go list ./... | grep -v vendor)
```

## Developing Chain Core

### Updating the schema with migrations

```
$ dumpschema
```

### Dependencies

To add or update a Go dependency, do the following:

Copy the code from `$GOPATH/src/x`
to `$CHAIN/vendor/x`. For example, to vendor the package
`github.com/kr/pretty`, run

```
$ mkdir -p $CHAIN/vendor/github.com/kr
$ rm -r $CHAIN/vendor/github.com/kr/pretty
$ cp -r $GOPATH/src/github.com/kr/pretty $CHAIN/vendor/github.com/kr/pretty
$ rm -rf $CHAIN/vendor/github.com/kr/pretty/.git
```

(Note: don’t put a trailing slash (`/`) on these paths.
It can change the behavior of cp and put the files
in the wrong place.)

In your commit message, include the commit hash of the upstream repo
for the dependency. (You can find this with `git rev-parse HEAD` in
the upstream repo.) Also, make sure the upstream working tree is clean.
(Check with `git status`.)

## Deploy Options

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/chain/chain/tree/main)

When Chain Core is deployed to a non-local host, all requests require authentication.
To generate a client access token on Heroku, run the following command:

```
$ heroku run -a <your-heroku-app> corectl create-token <token-name>
<token-name>:<your-token>
```

## License

Chain Core Developer Edition is licensed under the terms of the [GNU 
Affero General Public License Version 3 (AGPL)](LICENSE).

The Chain Java SDK (`/sdk/java`) is licensed under the terms of the 
[Apache License Version 2.0](sdk/java/LICENSE).
