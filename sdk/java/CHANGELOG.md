# Chain Java SDK changelog

This changelog is for the 1.2 branch of the Java SDK. Older versions:

- [1.1](https://github.com/chain/chain/blob/1.1-stable/sdk/java/CHANGELOG.md)
- [1.0](https://github.com/chain/chain/blob/1.0-stable/sdk/java/CHANGELOG.md)

## 1.2.0 (May 12, 2017)

This is a *minor version* release that includes breaking changes and new features. Before upgrading your SDK, please review a [a full summary of what's new in Chain Core 1.2](https://chain.com/docs/1.2/core/reference/changelog#1.2.0).

### Breaking changes

#### Control programs removed in favor of receivers

To perform cross-core transfers, you should use instances of `Receiver` instead of `ControlProgram`, which has been removed. The pattern for using receivers is nearly identical to using control programs. The key differences are:

- Receivers are created via `Account.createReceiver`, rather than a static method on the `Receiver` class.
- The `Transaction.Action.ControlWithProgram` transaction builder action has been replaced with `Transaction.Action.ControlWithReceiver`. When performing cross-core transfers, counterparties should not exchange raw control program strings. Instead, they should use `Receiver#toJson` to serialize the receiver objects, and then exchange the serialized receivers. See the [documentation](https://chain.com/docs/1.2/core/build-applications/transaction-basics#between-two-chain-cores) for examples.

Note that control programs still exist at the protocol level, and are exposed in transaction outputs in their raw form. However, they are no longer first-class objects in our SDK interfaces.

#### Tuple-style output identifiers

Chain Core 1.1 introduced a new pattern of identifying transaction outputs using a single unique identifier. This deprecated the 1.0-style identifier, which used a tuple of transaction ID and output position.

With Chain Core 1.2, the deprecated tuple identifier has been removed. This has two interface-level effects:

- `Transaction.Input` no longer has a `spentOutput` property. Use `spentOutputId` instead.
- `Transaction.Action.SpentAccountUnspentOutput` no longer has `setTransactionId` and `setPosition` methods. Use `setUnspentOutput` or `setOutputId` instead. See the [documentation](https://chain.com/docs/1.2/core/build-applications/unspent-outputs#spend-unspent-outputs) for examples.

### New features

#### Authentication and authorization

The `AccessToken` and `AuthorizationGrant` classes have been added to enable programmatic control over authentication and authorization of requests made to Chain Core. This lets you generate credentials that have well-scoped access to different parts of the Chain Core API.

See the [documentation](https://chain.com/docs/1.2/core/learn-more/authentication-and-authorization) for code samples, as well as a detailed introduction to authentication and authorization in Chain Core.

#### TLS configuration

The SDK now features flexible TLS configuration, allowing you to accept server certificates outside of the public X.509 PKI, and to authenticate requests to Chain Core using client X.509 certificates.

See the [documentation](https://chain.com/docs/1.2/core/learn-more/mutual-tls-auth) for specific instructions.

## 1.2.0rc2 (May 4, 2017)

Pre-release version.

## 1.2.0rc1

Unpublished pre-release version.

