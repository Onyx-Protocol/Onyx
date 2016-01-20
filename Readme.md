Chain 🍭

## Getting Started

Get and and compile the source:

	$ git clone https://github.com/chain-engineering/chain $GOPATH/src/chain
	$ cd $GOPATH/src/chain
	$ go install ./cmd/...

Create a development database:

	$ createdb api

## Testing

    $ createdb api-test
    $ go test $(go list ./... | grep -v vendor)

## Updating the schema with migrations

First, restore your database to the current version of `api/appdb/schema.sql`:

	$ dropdb api
	$ createdb api
	$ psql api < api/appdb/schema.sql

Next, run any migrations:

	$ psql api < migrations/your-migration.sql
	$ ...

Finally, dump the database schema, filtering any extension statements:

	$ pg_dump -sOx api | grep -v "CREATE EXTENSION" | grep -v "COMMENT ON EXTENSION" > api/appdb/schema.sql
