# Download Chain Core SDKs

Client libraries for Chain Core are available for the following platforms:

- [Java](#java)
- [Node.js](#node-js)
- [Ruby](#ruby)

To ensure compatibility between your application and Chain Core, choose an SDK version whose major and minor components (`major.minor.x`) match your version of Chain Core.

## Java

The Java SDK is available via [Maven](https://search.maven.org/#search%7Cga%7C1%7Cchain-sdk-java). Java 7 or greater is required.

**Maven** users should add the following to `pom.xml`:

```
<dependencies>
  <dependency>
    <groupId>com.chain</groupId>
    <artifactId>chain-sdk-java</artifactId>
    <version>1.0.1</version>
  </dependency>
</dependencies>
```

**Gradle** users should add the following to `build.gradle`:

```
compile 'com.chain:chain-sdk-java:1.0.1'
```

You can also [download the JAR](https://search.maven.org/remotecontent?filepath=com/chain/chain-sdk-java/1.0.1/chain-sdk-java-1.0.1.jar) as a binary.

## Node.js

The Chain Node.js SDK is available [via npm](https://www.npmjs.com/package/chain-sdk). Node 4 or greater is required.

To install, run the following command from your project directory:

```
npm install --save chain-sdk@1.0.2
```

## Ruby

The Ruby SDK is available [via Rubygems](https://rubygems.org/gems/chain-sdk). Ruby 2.1 or greater is required.

To install, add the following to your `Gemfile`:

```
gem 'chain-sdk', '~> 1.0.1', require: 'chain'
```
