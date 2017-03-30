# Package tags and branches

## List of versioned packages

- `chain-core-server`: `cored` and associated utilities. Previously known as `cmd.cored`.
- `chain-enclave`: HSM utilities, such as `signerd`.
- `chain-core-docker`: Docker image. Previously known as `docker.de`.
- `chain-core-mac`: Mac app. Previously known as `installer.mac`.
- `chain-core-windows`: Windows installer. Previously known as `installer.windows`.
- `sdk-java`
- `sdk-node`
- `sdk-ruby`

## Version branches

Every major/minor version pair will have its own branch named `<major>-<minor>-stable`, e.g. `1.1-stable`.

Version branches act as the merge base for point release updates across all packages, and should start with the commit corresponding to `chain-core-server-<major>.<minor>.0`.<sup>1</sup>

Updates to the version branches should be as conservative as possible. We should ensure that the tip of each version branch maintains cross-compatibility across packages, per our [versioning scheme](../core/reference/versioning.md).

The standards for including a bug fix in a point release of Chain Core Server are:

- the bug has no workaround
- the bug is likely to be hit (e.g. >1% probability)
- the bug compromises important functionality (i.e. not cosmetic, not minor)

If a bug meets all three of those standards, it is a good candidate to fix in a point release. Different products might decide to apply different standards for their point releases.

<sup>1</sup> Most package releases in 1.0.x predate this scheme. While the `1.0-stable` reflects the state-of-the-art in 1.0.x, it is not the merge base for older releases for that version family.

## Release tags

Every package release has a **tag** that specifies the package name and its major, minor, and build versions, e.g. `chain-core-server-1.1.0`.

Naturally, release tags should live on their relevant version branches, e.g. `chain-core-server-1.1.1` should be on the `1.1-stable` branch.
