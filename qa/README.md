# Chain Core QA Suite
All tests require proper credentials to a Chain Core and a copy of the Chain SDK.

### Build test suite

Install mvn:

    $ brew install mvn

Download and build the [Chain Java SDK](https://github.com/chain-engineering/chain-sdk-java).

Add the jar to your local maven repository:

    $ mvn install:install-file -Dfile=relative/path/to/jar -DpomFile=relative/path/to/pom.xml

Build the tests:

    $ cd $CHAIN/qa
    $ mvn package

### Run single core tests
> These tests exercise various, functional flows available within Chain Core.

Run tests

    $ CHAIN_API_URL=<auth_url> java -ea -cp path/to/chain-core-qa.jar chain.qa.baseline.singlecore.Main
