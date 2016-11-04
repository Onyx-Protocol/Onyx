# Software Versioning

* [Introduction](#introduction)
* [Version format](#version-format)
* [Build version](#build-version)
* [Compatibility](#compatibility)
* [Examples](#examples)
* [Alternatives considered](#alternatives-considered)

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
* as a dependency on another package (e.g. fix in a Chain Core needs corresponding Mac and Windows executables)

We need a consistent, yet simple, versioning scheme that allows both individual updates as well as grouped updates: from major feature releases to smaller, maintenance releases.

We do not want to formally commit to specific semantics of the version numbers, such as [SemVer](http://semver.org), but we need to keep all numbers in order, so it’s easier to navigate them.

## Version format

Use the following version format:

    <major>.<minor>.<package>

**Major** and **minor** versions are reserved for the entire product updates usually affecting several packages. Either version can be bumped based on the significance of the update. Individual packages must not use either **major** or **minor** version on their own without synchronization with the entire product line, but instead should use **package** version.

**Package** version must be formatted as dates in the following format: `YYYYMMDD` or omitted entirely. **Major** and **minor** versions are plain integers.

Examples:

* 1.0
* 1.0.20161024
* 1.7.20170119
* 2.3.20170601

## Package version

The reason to use dates for **package** versions is to avoid frequent collisions across different software packages that may convey non-existant correlation. E.g. if at some point Java SDK and Mac app both have version `1.3.2`, some may be confused by the identical version. 

On the other hand, dependent releases usually happen on the same day and therefore most likely will have the identical versions, indicating the correlation truthfully. E.g. if Chain Core adds a minor fix in the API, all SDKs may be updated with the same new build number: `1.0.161205`.

## Version format rationale

We have considered non-numerical values to separate package version further from the product versions, but these may cause undesireable issues with the ecosystem. For instance, rubygems does not want to have non-integers in the X.Y.Z format. And Sparkle updater on macOS sorts versions lexicographically when determining if the available update should be downloaded at all. 

## Compatibility

We do not commit to any correspondance between versions and compatibility. In the early development phase we will have to break things and that may happen in any version increment: major, minor or even the package one. As we polish our product, our release notes will eventually replace instructions to fix the code by more gentle deprecation notices.

Do not expect this guide to cover all imaginable scenarios, use common sense and the context to decide the versioning in unusual situations. We may even decide in the future to tweak versioning scheme from `A.B.YYMMDD` format to something else.

## Protocol versions

Chain Protocol versions are not covered by this document because it is imaginable that significant product upgrades from 1.4 to 1.5 would still use protocol 1. Some updates will correspond with protocol changes.

## Internal versions

Data structures and APIs may have their own internal version tags that are not part of the public product versioning. It is conceivable that Chain Core 1.4 will handle API version 9, block version 5 and transaction version 3.

See, for example, [Blockchain Extensibility](../protocol/papers/blockchain-extensibility.md) paper that describes versioning of the blockchain data structures.

## Build version

We will call a **build version** a git commit ID from which a package is built. It is not intended for front-and-center presentation, but can be available in the package information file for debugging purposes. 

For instance, macOS app may show the build version in the logs and About window. Java SDK may expose it as a `Chain.BUILD_VERSION` constant accessible by the application.

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
* **Java SDK 1.1**
* **Chain Core for Mac 1.1**
* **Chain Core Windows Installer 1.1**

Note that SDK does not need to be updated in this release since dashboard changes do not have a corresponding SDK support.
But SDK version will be bumped anyway with release notes say "no changes" to match the rest of the product versions.

### Case 4: two releases in one day

This will rarely happen, so we can simply increment the day number, "borrowing" the version number from tomorrow. We may even "overflow the month" by going from 20170131 to 20170132, if needed :) That won't be pretty, but that will not happen often either. If it does, then we can rethink the versioning scheme and add an additional number.


## Alternatives considered

### 1. Fully independent versioning

This potentially creates a messy environment where Chain Core 1.2.3 is packaged in a Mac app 3.5 with a Java SDK 2.1.4 and Ruby SDK 1.7.8.

### 2. Only major version is shared

The situation is a little less messy, but still: Chain Core 1.2.3 is packaged in a Mac app 1.3.5 with a Java SDK 1.1.4 and Ruby SDK 1.7.8.

### 3. All versions are synchronized

When one package is updated, all others are updated too with a version bump. 

Downsides: slower and more error-prone release process and large amount of noise in release notes everywhere.

### 4. Package version is an integer, independently updated

This will cause frequent confusing coincidences like macOS app has version 1.2.3 and Java SDK has version 1.2.3.

### 5. Package version is a singleton integer

Package version is a single integer, incremented by each update of any package. So after mac app is updated from 1.0.0 to 1.0.1, Java SDK afterwards can only be updated from 1.0.0 to 1.0.2. This avoid confusion caused by matching version numbers, but introduces another confusion about the gaps in the version numbers of each individual package. And the more packages we have, the bigger the gaps will be. In contrast, the date-based versions do not have expectation to not have gaps and naturally allow avoiding coinciding version numbers.


