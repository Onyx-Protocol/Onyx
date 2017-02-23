# Java SDK release guide

## Install dependencies

To prepare for this, ensure you have the following dependencies installed and working:

- [JDK 1.8](http://www.oracle.com/technetwork/java/javase/downloads/jdk8-downloads-2133151.html)
- Maven: `brew install maven`
- GPG: `brew install gpg`

## Gather credentials

Collect some credential information from Team 1Password:

- The Sonatype Jira **username** and **password** are located in the "Engineering" Team 1Password vault. Maven will use these to authenticate with the Central Repository, where the downloadable artifacts will be hosted.
- We use a GPG key to sign Maven artifacts. The **key ID**, **private key**, and **private key password** are located in the "Sensitive" Team 1Password vault. You'll probably need to bug a team lead to get access to this vault.

Remember to [import the GPG private key](http://irtfweb.ifa.hawaii.edu/~lockhart/gpg/) into your keychain.

## Setup Maven authentication and signing

Update your Maven settings in `$HOME/.m2/settings.xml` so it's ready to perform signing and uploading to Maven. Here's an example `settings.xml`:

```
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0
                      https://maven.apache.org/xsd/settings-1.0.0.xsd">

  <profiles>
    <profile>
      <id>gpg</id>
      <activation>
        <activeByDefault>true</activeByDefault>
      </activation>
      <properties>
        <gpg.keyname><!-- GPG KEY ID --></gpg.keyname>
        <gpg.passphrase><!-- PRIVATE KEY PASSWORD --></gpg.passphrase>
      </properties>
    </profile>
  </profiles>

  <servers>
    <server>
      <id>ossrh</id>
      <username><!-- SONATYPE JIRA USERNAME --></username>
      <password><!-- SONATYPE JIRA PASSWORD --></password>
    </server>
  </servers>
</settings>
```

## Increment the release version

1. Verify with the relevant parties that you really want to releas a new version.
1. Run the Java integration tests: `$CHAIN/bin/run-tests`
1. Update the `<version>` element in `pom.xml` with the appropriate version string.
1. Commit this update to Git, and tag the commit with `sdk.java-<version string>`.

## Deploy

1. Navigate to the Java SDK: `cd $CHAIN/sdk/java`
1. Build the artifacts and deploy to the Central Repository: `mvn clean deploy -DskipTests=true`
