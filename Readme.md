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

    $ go test $(go list ./... | grep -v vendor)

## Updating the schema with migrations

First, drop and recreate your database:

	$ dropdb api
	$ createdb api

Next, run all migrations, including your new migrations:

	$ migratedb -d postgres:///api?sslmode=disable
	$ ...

Finally, dump the database schema, filtering any extension statements:

	$ pg_dump -sOx api | grep -v "CREATE EXTENSION" | grep -v "COMMENT ON EXTENSION" > core/appdb/schema.sql

## Provisioning

First, make sure the following commands have been installed on your local machine:

- `api-add-user`
- `appenv`
- `migratedb`

From #devlog, provision the AWS resources:

	/provision api <target>

From your local machine, check out your desired branch for the `chain` project, and run database migrations:

	$ migratedb -t <target>

From #devlog, build and deploy the API server:

	/build [-t <git-branch>] api
	/deploy [-t <build-tag>] api <target>

From your local machine, create an API user:

	$ DB_URL=`appenv -t <target> DB_URL` api-add-user <email> <password>

From your local machine, create an API project and make the new user an admin:

	$ psql `appenv -t <target> DB_URL`
	api=# -- create a project
	api=# INSERT INTO projects (name) VALUES ('<project-name>'');
	api=# -- get the project ID
	api=# SELECT id FROM projects;
	api=# -- get your user ID
	api=# SELECT id FROM users;
	api=# -- make yourself an admin of the project
	api=# INSERT INTO members (project_id, user_id, role) VALUES ('<project-id>', '<user-id>', 'admin');

Finally, try logging into the dashboard at `https://<target>.chain.com`.

##### Provisioning TODO:

- Commandline tool to create projects
- Commandline tool to add members to projects
- `/provision` should automatically migrate and deploy given a specific git ref, defaulting to `main`.
