# Package tags and branches

## List of versioned packages

- `chain-core-server`: `cored` and associated utilities. Previously known as `cmd.cored`.
- `chain-enclave`: HSM utilities, such as `signerd`.
- `docs`: Markdown documentation, not including SDK-specific docs.
- `chain-core-docker`: Docker image. Previously known as `docker.de`.
- `chain-core-mac`: Mac app. Previously known as `installer.mac`.
- `chain-core-windows`: Windows installer. Previously known as `installer.windows`.
- `sdk-java`
- `sdk-node`
- `sdk-ruby`

## Version branches

Every major/minor version pair will have its own branch named `<major>-<minor>.x`, e.g:

- `1.0-stable`
- `1.1-stable`

Version branches act as the merge base for point release updates across all packages, and should start with the commit corresponding to `chain-core-server-<major>.<minor>.0`.<sup>1</sup>

Updates to the version branches should be as conservative as possible. We should ensure that the tip of each version branch maintains cross-compatibility across packages, per our [versioning scheme](../core/reference/versioning.md).

<sup>1</sup> `1.0-stable` is an exception, since it precedes this branching scheme.

## Release tags

Every package release has a **tag** that specifies the package name and its major, minor, and build versions, e.g.:

- `chain-core-server-1.1.0`
- `sdk-ruby-1.0.2`

Naturally, release tags should live on their relevant version branches, e.g.:

- `chain-core-server-1.1.1` is on the `1.1.x` branch
- `sdk-ruby-1.0.2` is on the `1.0.x` branch
