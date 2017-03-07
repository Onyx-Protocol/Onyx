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

Every major/minor version pair will have its own branch named `<major>-<minor>-stable`, e.g. `1.1-stable`.<sup>1</sup>

Version branches act as the merge base for point release updates across all packages, and should start with the commit corresponding to `chain-core-server-<major>.<minor>.0`.

Updates to the version branches should be as conservative as possible. We should ensure that the tip of each version branch maintains cross-compatibility across packages, per our [versioning scheme](../core/reference/versioning.md).

<sup>1</sup> Version 1.0 predates this scheme, so there is no `1.0-stable` branch. To make updates to 1.0, please use the 1.0 package-specific release tags.

## Release tags

Every package release has a **tag** that specifies the package name and its major, minor, and build versions, e.g. `chain-core-server-1.1.0`.

Naturally, release tags should live on their relevant version branches, e.g. `chain-core-server-1.1.1` should be on the `1.1-stable` branch.
