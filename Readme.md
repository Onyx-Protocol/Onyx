Chain 🍭

## Getting Started

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/chain/chain/tree/main)

First, make sure you have the dependencies installed:

* [Go](https://golang.org/doc/install), with $GOPATH set to your preferred directory
* Postgres (we suggest [Postgres.app](http://postgresapp.com/)), along with the [command line tools](http://postgresapp.com/documentation/cli-tools.html)
* [protoc](https://github.com/google/protobuf#protocol-compiler-installation), if you need to compile protos

### Environment

Set the `CHAIN` environment variable, in `.profile` in your home
directory, to point to the root of the Chain source code repo:

	export CHAIN
	CHAIN=$GOPATH/src/chain

You should also add `$CHAIN/bin` to your path (as well as `$GOPATH/bin`, if it isn't already):

	export PATH=$GOPATH/bin:$CHAIN/bin:$PATH

You might want to open a new Terminal window to pick up the change.

### Source Code

Get and and compile the source:

	$ git clone https://github.com/chain-engineering/chain $CHAIN
	$ cd $CHAIN
	$ go install ./cmd/...

Create a development database:

	$ createdb core

## Testing

    $ go test $(go list ./... | grep -v vendor)

## Updating the schema with migrations

	$ dumpschema

## Provisioning

First, make sure the following commands have been installed on your local machine:

	$ go install chain/cmd/{appenv,corectl,migratedb}

From #devlog, provision the AWS resources:

	/provision api <target>

From your local machine, check out your desired branch for the `chain` project, and run database migrations:

	$ migratedb -t <target>

From #devlog, build and deploy the Core server:

	/build [-t <git-branch>] api
	/deploy [-t <build-tag>] api <target>

Finally, try logging into the dashboard at `https://<target>.chain.com`.

##### Provisioning TODO:

- Commandline tool to create projects
- Commandline tool to add members to projects
- `/provision` should automatically migrate and deploy given a specific git ref, defaulting to `main`.
