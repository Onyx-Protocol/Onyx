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

**Build** version must be formatted as dates in the following format: `YYYYMMDD` or omitted entirely. **Major** and **minor** versions are plain integers.

Examples:

* 1.0
* 1.0.20161024
* 1.7.20170119
* 2.3.20170601

Compatibility notes must be specified separately. See [Blockchain Extensibility](../protocol/papers/blockchain-extensibility.md) document for details of blockchain compatibility. SDKs should deprecate old APIs and introduce new ones smoothly and may change entirely with major product releases.

## Build version

The reason to use dates for **build** versions is to avoid frequent collisions across different software pieces that may convey non-existant correlation. E.g. if at some point Java SDK and Mac app both have version `1.3.2`, some may be confused by the identical version. 

On the other hand, dependent releases usually happen on the same day and therefore most likely will have the identical versions, indicating the correlation truthfully. E.g. if Chain Core adds a minor fix in the API, all SDKs may be updated with the same new build number: `1.0.161205`.

## Compatibility

We should strive to keep expectation that minor version updates do not break backward compatibility. E.g. if we need to introduce a completely new SDK, we should bump the major version, or rename the SDK. If we need to retire/improve all APIs, we should add a compatibility layer and diagnostic messages to the SDK without simply removing them.

Do not expect this guide to cover all imaginable scenarios, use common sense and the context to decide the versioning in unusual situations. Also, even in such cases versioning guidelines can be seen not as an unfortunate barrier, but as a useful constraint that may guide us to a better positioning of the product updates and help avoid 


## Examples

Suppose, we have the following versions currently shipping:

* Chain Core 1.0
* Java SDK 1.0
* Chain Core for Mac 1.0
* Chain Core Windows Installer 1.0

### Case 1: a bugfix in the SDK

Bump the build version of the SDK, other versions remain unchanged:

* Chain Core 1.0
* **Java SDK 1.0.20161027**
* Chain Core for Mac 1.0
* Chain Core Windows Installer 1.0

### Case 2: a bugfix in the API

Bump the build version of Chain Core and SDK, update packages depending on Chain Core.

* **Chain Core 1.0.20161101**
* **Java SDK 1.0.20161101**
* **Chain Core for Mac 1.0.20161101**
* **Chain Core Windows Installer 1.0.20161101**

### Case 3: a new dashboard feature is released

Bump the minor version of Chain Core, update packages depending on Chain Core.

* **Chain Core 1.1**
* Java SDK 1.0.20161101
* **Chain Core for Mac 1.1**
* **Chain Core Windows Installer 1.1**

Note that SDK is not updated in this release since dashboard changes do not have corresponding SDK support.
SDK version will be bumped to match the rest of the products’ versions when it’s updated next time.

### Case 4: a new feature added to Chain Core API

Bump the minor version of Chain Core and SDK, update packages depending on Chain Core.

* **Chain Core 1.2**
* **Java SDK 1.2**
* **Chain Core for Mac 1.2**
* **Chain Core Windows Installer 1.2**

Note, that with this release, SDK version skipped 1.1 and went directly to 1.2 to synchronize with the Chain Core version (similar to how lamport timestamps work ;-). 

If we just implemented a bugfix in SDK alone, then we’d also bump the major and minor versions to 1.1 to catch up with the rest of the product, and added a build version to it (instead of just bumping the build version on 1.0):

* Chain Core 1.1
* **Java SDK 1.1.20161108**
* Chain Core for Mac 1.1
* Chain Core Windows Installer 1.1


## Alternatives considered

### 1. Fully independent versioning

This potentially creates a messy environment where Chain Core 1.2.3 is packaged in a Mac app 3.5 with a Java SDK 2.1.4 and Ruby SDK 1.7.8.

### 2. Only major version is shared

The situation is a little less messy, but still: Chain Core 1.2.3 is packaged in a Mac app 1.3.5 with a Java SDK 1.1.4 and Ruby SDK 1.7.8.

### 3. All versions are synchronized

When one piece of software is updated, all others are updated too with a version bump. 

Downsides: slower and more error-prone release process and large amount of noise in release notes everywhere.


