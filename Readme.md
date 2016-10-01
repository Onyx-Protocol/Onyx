Chain 🍭

## Getting Started

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/chain/chain/tree/main)

## Contributing

Chain has adopted the code of conduct defined by the Contributor Covenant. It can be read in full [here](https://github.com/chain/chain/blob/main/CODE_OF_CONDUCT.md).

### Dependencies

* [Go](https://golang.org/doc/install) version 1.7, with $GOPATH set to your
  preferred directory
* Postgres (we suggest [Postgres.app](http://postgresapp.com/)),
  along with the [command line
  tools](http://postgresapp.com/documentation/cli-tools.html)
* [protoc](https://github.com/google/protobuf#protocol-compiler-installation),
  if you need to compile protos

### Environment

Set the `CHAIN` environment variable, in `.profile` in your home
directory, to point to the root of the Chain source code repo:

	export CHAIN=$GOPATH/src/chain

You should also add `$CHAIN/bin` to your path (as well as
`$GOPATH/bin`, if it isn't already):

	PATH=$GOPATH/bin:$CHAIN/bin:$PATH

You might want to open a new terminal window to pick up the change.

### Installation

Build and install from source:

	$ git clone https://github.com/chain/chain $CHAIN
	$ cd $CHAIN
	$ go install ./cmd/...

Set up the database:

	$ createdb core

Start Chain Core:

	$ cored

Access the dashboard:

	$ open http://localhost:8080/

Run tests:

    $ go test $(go list ./... | grep -v vendor)

## Updating the schema with migrations

	$ dumpschema

## Provisioning

First, make sure the following commands have been installed on
your local machine:

	$ go install chain/cmd/{appenv,corectl,migratedb}

From #devlog, provision the AWS resources:

	/provision api <target>

From your local machine, check out your desired branch for the
`chain` project, and run database migrations:

	$ migratedb -t <target>

Then create an initial block:

	$ DB_URL=postgres://... corectl init 1 [key]

From #devlog, build and deploy the Core server:

	/build [-t <git-branch>] api
	/deploy [-t <build-tag>] api <target>

Finally, try logging into the dashboard at `https://<target>.chain.com`.

##### Provisioning TODO:

- Commandline tool to create projects
- Commandline tool to add members to projects
- `/provision` should automatically migrate and deploy given a
  specific git ref, defaulting to `main`.

## Dependencies

To add or update a Go dependency, do the following:

Copy the code from `$GOPATH/src/x`
to `$CHAIN/vendor/x`. For example, to vendor the package
`github.com/kr/pretty`, run

	$ mkdir -p $CHAIN/vendor/github.com/kr
	$ rm -r $CHAIN/vendor/github.com/kr/pretty
	$ cp -r $GOPATH/src/github.com/kr/pretty $CHAIN/vendor/github.com/kr/pretty
	$ rm -rf $CHAIN/vendor/github.com/kr/pretty/.git

(Note: don't put a trailing slash (`/`) on these paths.
It can change the behavior of cp and put the files
in the wrong place.)

In your commit message, include the commit hash of the upstream repo
for the dependency. (You can find this with `git rev-parse HEAD` in
the upstream repo.) Also, make sure the upstream working tree is clean.
(Check with `git status`.)

### License

Chain Core Developer Edition is licensed under the terms of the [GNU 
Affero General Public License Version 3 (AGPL)](LICENSE).

The Chain Java SDK (`/sdk/java`) is licensed under the terms of the 
[Apache License Version 2.0](sdk/java/LICENSE).

