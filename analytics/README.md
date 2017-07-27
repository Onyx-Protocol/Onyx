# Chain Analytics

Chain Analytics is a separate java service for importing transactions into an Oracle database. It reads transactions through a Chain Core transaction feed, inserting them one-by-one into an Oracle database.

## Setup

These instructions assume you already have Oracle and Chain Core running locally.

**Compile and package**

```
mvn install
```

**Set environment variables**

The Chain Analytics program requires a few environment variables:
* `CHAIN_URL`: the URL of Chain Core
* `CHAIN_ACCESS_TOKEN`: a Chain Core access token
* `DATABASE_URL`: the URL of Chain Core and the DSN of the Oracle database. Replace `<oracle-username>` and `<oracle-password>` with your Oracle username and password. If you're using the Oracle Developer DB VM, you should be able to use `system` and `oracle`.

```
export CHAIN_URL="http://localhost:1999"
export CHAIN_ACCESS_TOKEN="<example>"
export DATABASE_URL="jdbc:oracle:thin:<oracle-username>/<oracle-password>@127.0.0.1:1521/orcl"
```

**Configure custom columns**

Chain Analytics allows you to build custom columns from transaction metadata. To adopt a configuration, run Chain Analytics with the arguments `migrate` and the path to your json configuration file. For an example configuration file, see src/main/resources/example-config.json.

```
# Migrate the Oracle database and configure the importer
# to index your reference data columns.
java -cp "target/analytics-1.0.1-jar-with-dependencies.jar" com.chain.analytics.Application migrate config.json
```

**Run the daemon**

To run the daemon and sync transactions, provide the argument `run`.

```
# Start syncing transactions.
java -cp "target/analytics-1.0.1-jar-with-dependencies.jar" com.chain.analytics.Application run
```

The daemon will automatically create a transaction feed and begin syncing transactions. Create a transaction in dashboard and verify that it's synced.
