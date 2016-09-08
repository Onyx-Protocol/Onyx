# Chain Core QA Suite
### Build jar

Install mvn:

    $ brew install mvn

Build the Chain Core SDK jar

Add the jar to your local maven repository:

    $ mvn org.apache.maven.plugins:maven-install-plugin:2.5.2:install-file -Dfile=relative/path/to/jar

Build the QA Suite:

    $ cd $CHAIN/qa
    $ mvn package

### Single core tests
> These tests exercise various, functional flows available within a single core network.

Your core url can be passed as the CHAIN_API_URL environment variable. All programs will default to http://localhost:8080

Run basic example

    $ java -cp path/to/chain-core-qa.jar com.chain_qa.BasicExample

Run multisig example

    $ java -cp path/to/chain-core-qa.jar com.chain_qa.MultiSigExample

Run reference data example

    $ java -cp path/to/chain-core-qa.jar com.chain_qa.ReferenceDataExample
