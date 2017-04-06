# Chain Analytics

Chain Analytics is a separate java service for importing transactions into an Oracle database. It reads transactions through a Chain Core transaction feed, inserting them one-by-one into an Oracle database.

## Setup

These instructions assume you already have Oracle and Chain Core running locally.

**Compile and package**

```
mvn install
```

**Run the daemon**

The daemon takes two environment variables: the URL of Chain Core and the DSN of the Oracle database. Replace `<oracle-username>` and `<oracle-password>` with your Oracle username and password. If you're using the Oracle Developer DB VM, you should be able to use `system` and `oracle`.

```
export CHAIN_URL="http://localhost:1999"
export DATABASE_URL="jdbc:oracle:thin:<oracle-username>/<oracle-password>@127.0.0.1:1521/orcl"
java -cp "target/analytics-1.0.0-jar-with-dependencies.jar" analytics.Application
```

The daemon will automatically initialize the Oracle database with a schema, create a transaction feed and begin syncing transactions. Create a transaction in dashboard and verify that it's synced.
