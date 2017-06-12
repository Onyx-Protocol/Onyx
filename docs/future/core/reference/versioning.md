# Versioning

## Summary

There are two distinct Chain versioning schemes:

* **core** versioning
* **network** versioning

Core versioning applies to the main pieces of software that compose, package, or directly interact with Chain Core. This includes, for example, the Chain Core server, the client SDK libraries, and the platform-specific installers.

Network versioning applies to the network API--the interface through which different Chain Cores communicate to exchange transactions and blocks. Any breaking changes to the network API interface or the data structures as defined by the Chain protocol are reflected by a change to the network version.

## Core Versioning

The various packages of the Chain Core suite each use a three-number versioning scheme: **x.y.z**. The first two numbers indicate compatibility between packages. Server software is compatible with clients that share the same major and minor version, or are one minor version behind. For example, if you are a running a version of the Chain Core server whose first two numbers are **1.1**, you can use a version of an SDK whose first two numbers are **1.0** or **1.1**.

Sometimes we make bugfixes and minor feature updates to individual software packages. When this occurs, we update the third number in the version string for that package only. This number doesn't affect compatibility with other packages. For example, you can safely use version **1.1.1** of the Chain Core Mac App with version **1.1.3** of the Java SDK.

### Scope

The core versioning scheme covers a suite of individual software packages, including servers, clients, and bundled releases.

Server packages include:

- `cored`, the Chain Core server daemon
- `signerd`, the HSM signing server daemon
- Thales Codesafe HSM firmware

Client packages include:

- `corectl`, a CLI utility for operating Chain Core
- Java SDK
- Node.js SDK
- Ruby SDK

Bundled releases include:

- Chain Core Mac app
- Chain Core Windows app
- Chain Core Docker image

### Format

Each release of a core software package is assigned a version string composed of three numbers separated by periods, such as **1.0.1**.

These numbers represent, in order:

- **Major version**: significant product changes
- **Minor version**: bugfixes, new features, deprecations, and breaking changes
- **Build version**: package-specific bugfixes and features

### Semantics

This scheme has semantics that are unique to Chain Core, despite superficial similarities to other versioning schemes. In particular, Chain Core does **not** use [Semantic Versioning](http://semver.org/).

#### Compatibility between packages

Software packages are mutually compatible if they share the same major and minor version. To accommodate a smooth upgrade flow, server packages are backward compatible with client packages that are one minor version behind. For example, if you are running version 1.1.x of the Chain Core server, you can use version 1.0.x or 1.1.x of the Java SDK.

#### Deprecations and breaking changes

Deprecations and breaking changes can occur when the minor version changes. Breaking changes in server packages will be preceded by a deprecation announcement and at least one minor version cycle of continued support.

Breaking changes may be introduced at any time in client packages (such as SDKs). However, since server packages are backward-compatible with the previous minor version of client packages, you can safely upgrade your server packages without breaking your application code.

#### Upgrading example

Chain Core is a quickly growing product, but the versioning scheme is designed to allow for a smooth upgrade process. To demonstrate this, here's an example of how to perform a minor-vesrion upgrade of Chain Core and the client SDK. Assume we're using Chain Core 1.1 and SDK 1.1, and wish to upgrade to 1.2.

1. Upgrade Chain Core to 1.2, the next minor version of server software. The versioning rules specify that it the new version is backward-compatible with 1.1 SDKs, so your application code should continue to work.
2. In a development environment, upgrade your SDKs to 1.2. Adjust your application code to account for any interface changes in the new SDK version.
3. Deploy your upgraded application code from the previous step.

## Network versioning

The network version is a single integer. All instances of Chain Core connecting to a network must have the same network version. The network version is determined by the Chain Core operating as the block generator.

The network version will be incremented each time there is a breaking change in the network API interface or a breaking change in the version of the Chain protocol being implemented.

### Breaking changes

An upgrade in network version constitutes a breaking change at the network level. Any upgrade to the network version will be included in a new package version of Chain Core software and documented as a breaking network change in the release notes.

## Notes

### Enterprise vs. developer editions

Chain Core Enterprise Edition is a superset of Developer Edition. The two editions share the same version space. For example, if a particular feature exists in version 1.1.x of Developer Edition, then it also exists in version 1.1.x of Enterprise Edition.

### Older software

Chain Core is a different product than older software released by Chain, such as the Chain Blockchain Platform and related SDKs, the Chain sandbox and related SDKs, or SDKs for the Chain Bitcoin API. Chain Core does **not** share the same version space as older products.
