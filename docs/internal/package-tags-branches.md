# Package tags and branches

## List of versioned packages

- `cmd.cored`
- `cmd.signerd`
- `docker.de`
- `docs`: Markdown documentation, not including SDK-specific docs.
- `installer.mac`
- `installer.windows`
- `sdk.java`
- `sdk.node`
- `sdk.ruby`

## Basic scheme

Every release has a **tag** that specifies the major, minor, and build versions, e.g.:

- `cmd.cored-1.1.0`
- `sdk.ruby-1.0.2`

If a point release is necessary for a package, we should create a new major-minor version **branch** e.g.:

- `cmd.cored-1.1.x`
- `sdk.ruby-1.0.x`

Note that there's a `.x` suffix to distinguish the branch name from the corresponding `.0` release.

Updates to these branches should be as conservative as possible.  Our [versioning scheme](../core/reference/versioning.md) is such that it should always be safe to deploy artifacts from the tip of each package branch.

Naturally, release tags should live on their relevant branches, e.g.:

- `cmd.cored-1.1.0` is on the `cmd.cored-1.1.x` branch
- `sdk.ruby-1.0.2` is on the `sdk.ruby-1.0.x` branch
