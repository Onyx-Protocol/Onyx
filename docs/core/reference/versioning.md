# Versioning

## Summary

There are three distinct Chain versioning schemes:

* **core** versioning
* **network** versioning
* **signer** versioning

Core versioning applies to the main pieces of software that comprise, package, or directly interact with Chain Core (excluding those that facilitate transaction and block signing). For example, the Chain Core Mac app and the Java SDK.

Network versioning applies to the network API - the interface through which different Chain Cores on a network communicate to exchange transactions and blocks. Any breaking changes to the network API interface or the data structures as defined by the  Chain protocol are reflected by a change to the network version.

Signer versioning applies to the pieces of software that operate or interact with HSMs to facilitate transaction and block signing. These versions will change infrequently.

## Core Versioning
The various packages of the Chain Core suite each use a three number versioning scheme - **x.y.z**. The first two numbers indicate compatibility between packages, with a tolerance of `+/-1`. For example, if you are a running a version of the Chain Core Mac app whose first two numbers are **1.1**, you can use a version of an SDK whose first two numbers are **1.0**, **1.1**, or **1.2**.

Sometimes we make bugfixes and minor feature updates to individual software packages. When this occurs, we update the third number in the version string for that package only. This number doesn't affect compatibility with other packages. For example, you can safely use version **1.1.1** of the Chain Core Mac App with version **1.1.3** of the Java SDK.

### Scope

The core versioning scheme covers a suite of individual software packages:

- `cored`, the Chain Core server daemon
- `corectl`, a CLI utility for operating Chain Core
- Client API SDKs, such as the Java and Ruby SDKs
- the Chain Core Mac app
- the Chain Core Windows app
- the Chain Core Docker image

### Format

Each release of a core software package is assigned a version string composed of three numbers separated by periods, such as **1.0.1**.

These numbers represent, in order:

- **Major version**: significant product changes
- **Minor version**: bugfixes, new features
- **Build version**: package-specific bugfixes and features

The major version is shared by all packages in the Chain Core suite. If there is a change in the major version, then there will be a new release of all packages in the suite.

The minor version is shared by all packages with a tolerance of `+/-1`.

The build version of a specific package may change independently of other packages.

### Semantics

This scheme has semantics that are unique to Chain Core, despite superficial similarities to other versioning schemes such as [Semantic Versioning](http://semver.org/).

#### Compatibility between packages
Software packages in the Chain Core suite are compatible if their major and minor versions are no more than one number apart. For example, if you are running version 1.1.x of the Chain Core server, you can use version 1.0.x, 1.1.x, or 1.2.x of the Java SDK.

#### Breaking changes and backward compatibility

The version string does **not** convey whether a particular release contains a breaking change or not. Breaking changes will be announced in advance and documented in release notes. In general, we will do our best to avoid breaking changes.

## Network versioning
The network version is a single, incremented integer. All instances of Chain Core connecting to a network must have the same network version. The network version is determined by the Chain Core operating as the block generator.

The network version will be incremented each time there is a breaking change in the network API interface or a breaking change in the version of the Chain protocol being implemented.

### Breaking changes
An upgrade in network version constitutes a breaking change at the network level. Any upgrade to the network version will be included in a new package version of Chain Core software and documented as a breaking network change in the release notes.

## Signer Versioning
The various packages of signing software each use a three number versioning scheme - **x.y.z**. The first two numbers indicate compatibility between packages, with a tolerance of `+/-1`. For example, if you are a running a version of `signerd` whose first two numbers are **1.1**,  you can use a version of the Thales Codesafe HSM firmware whose first two numbers are **1.0**, **1.1**, or **1.2**.

Sometimes we make bugfixes and minor feature updates to individual software packages. When this occurs, we update the third number in the version string for that package only. This number doesn't affect compatibility with other packages. For example, you can safely use version **1.1.1** of `signerd` with version **1.1.3** of the Thales Codesafe HSM firmware.

### Scope
The signer versioning scheme covers the following packages:

- Chain Core MockHSM
- `signerd`, the HSM signing server daemon
- Thales Codesafe HSM firmware

### Format

Each release of a signer software package is assigned a version string composed of three numbers separated by periods, such as **1.0.1**.

These numbers represent, in order:

- **Major version**: significant product changes
- **Minor version**: bugfixes, new features
- **Build version**: package-specific bugfixes and features

The major version is shared by all signer packages. If there is a change in the major version, then there will be a new release of all signer software packages.

The minor version is shared by all signer packages with a tolerance of `+/-1`.

The build version of a specific package may change independently of other packages.

### Semantics

This scheme has semantics that are unique to Chain signer software packages, despite superficial similarities to other versioning schemes such as [Semantic Versioning](http://semver.org/).

#### Compatibility between packages
Signer software packages are compatible if their major and minor versions are no more than one number apart. For example, if you are running version 1.1.x of `signerd`, you can use version 1.0.x, 1.1.x, or 1.2.x of the Thales Codesafe HSM firmware.

#### Breaking changes and backward compatibility

The version string does **not** convey whether a particular release contains a breaking change or not. Breaking changes will be announced in advance and documented in release notes. In general, we will do our best to avoid breaking changes.

## Notes

### Enterprise vs. developer editions

Chain Core Enterprise Edition is a superset of Developer Edition. The two editions share the same version space. For example, if a particular feature exists in version 1.1.x of Developer Edition, then it also exists in version 1.1.x of Enterprise Edition.

### Older software

Chain Core is a different product than older software released by Chain, such as the Chain Blockchain Platform and related SDKs, the Chain sandbox and related SDKs, or SDKs for the Chain Bitcoin API. Chain Core does **not** share the same version space as older products.
