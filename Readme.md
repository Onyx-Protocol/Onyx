Chain 🍭

## Getting Started

### Environment

Set the `CHAIN` environment variable, in `.profile` in your home
directory, to point to the root of the Chain source code repo:

	export CHAIN
	CHAIN=$GOPATH/src/chain

You might want to open a new Terminal window to pick up the change.

### Source Code

Get and and compile the source:

	$ git clone https://github.com/chain-engineering/chain $CHAIN
	$ cd $CHAIN
	$ go install ./cmd/...

Create a development database:

	$ createdb api

## Testing

    $ createdb api-test
    $ go test $(go list ./... | grep -v vendor)

## Updating the schema with migrations

First, drop and recreate your database:

	$ dropdb api
	$ createdb api

Next, run all migrations, including your new migrations:

	$ migratedb -d postgres:///api?sslmode=disable
	$ ...

Finally, dump the database schema, filtering any extension statements:

	$ pg_dump -sOx api | grep -v "CREATE EXTENSION" | grep -v "COMMENT ON EXTENSION" > api/appdb/schema.sql
