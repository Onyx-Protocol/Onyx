# Chain Node.js SDK

This changelog is for the 1.2 branch of the Node.js SDK. Older versions:

- [1.1](https://github.com/chain/chain/blob//1.1-stable/sdk/node/CHANGELOG.md)
- [1.0](https://github.com/chain/chain/blob/1.0-stable/sdk/node/CHANGELOG.md)

## 1.2.0 (May 12, 2017)

This is a _minor version_ release that includes breaking changes and new features. Before upgrading your SDK, please review a [a full summary of what's new in Chain Core 1.2](https://chain.com/docs/1.2/core/reference/changelog#1.2.0).

### Breaking changes

#### Control programs removed in favor of receivers

To perform cross-core transfers, you should use receivers instead of control program. The pattern for using receivers is nearly identical to using control programs. The key differences are:

- Receivers are created via the following pattern:

    ```
    client.accounts.createReceiver(accountId: '...').then(...)
    ```

    This differs slightly to the removed `client.control_programs.create()` pattern.
- When performing cross-core transfers, counterparties should not exchange raw control program strings. Instead, they should use `JSON.stringify(receiver)` to serialize recevier objects, and then exchange the serialized receivers. See the [documentation](https://chain.com/docs/1.2/core/build-applications/transaction-basics#between-two-chain-cores) for examples.

Note that control programs still exist at the protocol level, and are exposed in transaction outputs in their raw form. However, they are no longer first-class objects in our SDK interfaces.

#### Tuple-style output identifiers

Chain Core 1.1 introduced a new pattern of identifying transaction outputs using a single unique identifier. This deprecated the 1.0-style identifier, which used a tuple of transaction ID and output position.

With Chain Core 1.2, the deprecated tuple identifier has been removed. This has two interface-level effects:

- Transaction inputs no longer contain a `spentOutput` property. Use `spentOutputId` instead.
- The `spendUnspentOutput` transaction builder method no longer accepts the `transactionId` and `position` parameters. Use `outputId` instead. See the [documentation](https://chain.com/docs/1.2/core/build-applications/unspent-outputs#spend-unspent-outputs) for examples.

## 1.2.0-rc.2 (May 4, 2017)

Pre-release version.

## 1.2.0-rc.1

Unpublished pre-release version.
