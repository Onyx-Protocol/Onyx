# Chain Java SDK

## Formatting
We adhere to the Google Java Style guide. To format code run:
```
$ jfmt --replace [files...]
```

## Build
```
$ mvn package
```
This will build a shaded jar (dependencies included with rewritten classpaths) and docs.
The shaded jar is named `chain-java-sdk-[version].jar` A jar sans dependencies is also created
with prefix `original-`. Docs can be found under `target/apidocs`.

## Run integration tests
Run the integration tests against a core listening on `http://localhost:1999`:
```
$ mvn integration-test
```

To set the core url use the `chain.api.url` system variable:
```
$ mvn -Dchain.api.url=http://localhost:8081 integration-test
```

## Write an integration test
Tests can be found in `src/test/java/com/chain/integration`. Suffix the class name with `Test` and add a `run()` method with the @Test annotation.

## License

The Chain Java SDK is licensed under the terms of the [Apache License Version 2.0](LICENSE).
