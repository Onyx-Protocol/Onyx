# Release Versions

## Introduction

We have several pieces of software:

* Chain Core itself
* SDKs (currently Java, with Ruby and Node.js in the roadmap)
* Mac app
* Windows installer
* Docker container

Each of these is going to be updated for one of three reasons:

* its own improvements (e.g., a bugfix in the SDK or a Mac app)
* improvements in a product as a whole (e.g. a new feature in Chain Core with updated SDK support)
* as a dependency on another piece (e.g. fix in a Chain Core needs corresponding Mac and Windows executables)

We need a consistent, yet simple, versioning scheme that allows both individual updates as well as grouped updates: from major feature releases to smaller, maintenance releases.

We probably do not want to formally commit to specific semantics of the version numbers, such as [SemVer](http://semver.org), but we need to keep all numbers in order, so it’s easier navigate them.

## Proposal

Use the following version format:

    <major>.<minor>.<build>

**Major** and **minor** versions are reserved for the entire product updates usually affecting several pieces of software. Either version can be bumped based on the significance of the update. Individual pieces of software must not use either **major** or **minor** version on their own without synchronization with the entire product line, but instead should use **build** version.

**Build** version must be formatted as date in the following format: `YYYYMMDD` or omitted entirely. **Major** and **minor** versions are plain integers.

Examples:

* 1.0
* 1.0.20161024
* 1.7.20170119
* 2.3.20170601

Compatibility notes must be specified separately. See [Blockchain Extensibility](../protocol/papers/blockchain-extensibility.md) document for details of blockchain compatibility. SDKs should deprecate old APIs and introduce new ones smoothly and may change entirely with major product releases.

## Examples

TBD.


## Alternatives considered

One option is to keep all versions synchronized: when one piece of software is updated, all others are updated too with a version bump. Downsides: slower and more error-prone release process and large amount of noise in release notes everywhere.

