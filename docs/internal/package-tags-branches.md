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

- `1.0.x`
- `1.1.x`

Version branches should be the target for point release updates across all packages. These updates should be as conservative as possible.  Our [versioning scheme](../core/reference/versioning.md) is such that it should always be safe to deploy artifacts from the tip of each package branch.

## Release tags

Every package release has a **tag** that specifies the package name and its major, minor, and build versions, e.g.:

- `chain-core-server-1.1.0`
- `sdk-ruby-1.0.2`

Naturally, release tags should live on their relevant version branches, e.g.:

- `chain-core-server-1.1.1` is on the `1.1.x` branch
- `sdk-ruby-1.0.2` is on the `1.0.x` branch
