# Chain Java SDK

## How to build

This will build a shaded jar (dependencies included with rewritten classpaths) and docs.
The shaded jar is named `chain-java-sdk-[version].jar` A jar sans dependencies is also created
with prefix `original-`. Docs can be found under `target/apidocs`.

```
$ mvn package
```
