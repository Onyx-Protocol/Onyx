# Chain Core QA Suite
What follows is a description of the processes used to assure quality for Chain Core. We employ three major types of tests: [Baseline](#base),  [Forced Error](#error) & [Performance](#perf). All tests require proper credentials to a Chain Core stack and a copy of the Chain SDK.

## <a name="base"></a>Baseline Testing [:leftwards_arrow_with_hook:](#chain-core-qa-suite)
> These tests exercise each SDK method in isolation as well as functionally testing the Chain Core API.

### Focus
- Testing all combinations of method arguments
- Testing boundary conditions for variable number arguments (i.e., arrays or objects)
- Testing default parameter assumptions (i.e., optional arguments)
- Assuring correct return tags and values
- Simulating common use case flows
- Simulating concurrent and batch transactions
- Observing/adhering to data dependencies between calls
- Observing correct state transitions between calls

### Run Test

Install mvn:

    $ brew install mvn

Download and build the [Chain Java SDK](https://github.com/chain-engineering/chain-sdk-java).

Add the jar to your maven local repository:

    $ mvn install:install-file -Dfile=relative/path/to/jar -DpomFile=relative/path/to/pom.xml

Run the test:

    $ cd $CHAIN/qa
    $ mvn package
    $ java -cp target/chain-core-qa.jar chain.qa.baseline.Main <authenticated url>

## <a name="error"></a>Forced Error Testing [:leftwards_arrow_with_hook:](#chain-core-qa-suite)
> These tests contain typical error scenarios such as: missing required arguments, incorrect arguments, exceeding maximum arguments limits, etc...

### Focus
- Assuring correct error occurrences
- Assuring correct error messages

## <a name="perf"></a>Performance Testing [:leftwards_arrow_with_hook:](#chain-core-qa-suite)
> These tests will benchmark the performance of our API endpoints. The main goal will be to approximate our systems maximum throughput (in requests per second).

Traffic will be simulated based on answers/approximations to following questions:

- What is our *average* throughput?
- What is our *peak* throughput?
- What is our throughput distribution by API endpoint?
