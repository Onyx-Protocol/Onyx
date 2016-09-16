# Chain Java SDK

## Build
```
$ mvn package
```
This will build a shaded jar (dependencies included with rewritten classpaths) and docs.
The shaded jar is named `chain-java-sdk-[version].jar` A jar sans dependencies is also created
with prefix `original-`. Docs can be found under `target/apidocs`.

## Run integration tests
Run the integration tests against a core listening on `http://localhost:8080`:
```
$ mvn integration-test
```

To set the core url use the `chain.api.url` system variable:
```
$ mvn -Dchain.api.url=http://localhost:8081 integration-test
```

## Write an integration test
Tests can be found in `src/test/java/com/chain/integration`. Suffix the class name with `Test` and add a `test()` method with the @Test annotation.
