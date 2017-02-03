# Download Chain Core SDKs

Chain Core SDKs enable your appplication to communicate with Chain Core.

## Java

The Java SDK is available via Maven, on Sonatype's [Central Repository](http://central.sonatype.org/). Simply add the following to your `pom.xml`:

```
<dependencies>
  <dependency>
    <groupId>com.chain</groupId>
    <artifactId>chain-sdk-java</artifactId>
    <version>1.0.1</version>
  </dependency>
</dependencies>
```

You can also [download the JAR](../../java/chain-sdk-latest.jar) as a binary.

## Ruby

The Ruby SDK is available [via Rubygems](https://rubygems.org/gems/chain-sdk). Make sure to use the most recent version whose major and minor components (`major.minor.x`) match your version of Chain Core. Ruby 2.1 or greater is required.

For most applications, you can simply add the following to your `Gemfile`:

```
gem 'chain-sdk', '~> 1.0.1', require: 'chain'
```

## Node.js

The Chain Node.js SDK is available [via npm](https://www.npmjs.com/package/chain-sdk). Make sure to use the most recent version whose major and minor components (major.minor.x) match your version of Chain Core. Node 4 or greater is required.

For most applications, you can simply add Chain to your `package.json` with the following command:

```
npm install --save chain-sdk@1.0.2
```
