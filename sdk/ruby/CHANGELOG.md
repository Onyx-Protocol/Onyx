# Chain Ruby SDK

This changelog is for the 1.2 branch of the Ruby SDK. Older versions:

- [1.1](https://github.com/chain/chain/blob//1.1-stable/sdk/ruby/CHANGELOG.md)
- [1.0](https://github.com/chain/chain/blob/1.0-stable/sdk/ruby/CHANGELOG.md)

## 1.2.0 (May 12, 2017)

This is a *minor version* release that includes breaking changes and new features. Before upgrading your SDK, please review a [a full summary of what's new in Chain Core 1.2](https://chain.com/docs/1.2/core/reference/changelog#1.2.0).

### Breaking changes

#### Control programs removed in favor of receivers

To perform cross-core transfers, you should use instances of `Chain::Receiver` instead of `Chain::ControlProgram`, which has been removed. The pattern for using receivers is nearly identical to using control programs. The key differences are:

- Receivers are created via the following pattern:

    ```
    client.accounts.create_receiver(account_id: '...')
    ```

    This differs somewhat to the removed `client.control_programs.create` pattern.
- The `control_with_program` transaction builder method has been replaced with `control_with_receiver`.
When performing cross-core transfers, counterparties should not exchange raw control program strings. Instead, they should use `Chain::Receiver#to_json` to serialize recevier objects, and then exchange the serialized receivers. See the [documentation](https://chain.com/docs/1.2/core/build-applications/transaction-basics#between-two-chain-cores) for examples.

Note that control programs still exist at the protocol level, and are exposed in transaction outputs in their raw form. However, they are no longer first-class objects in our SDK interfaces.

#### Tuple-style output identifiers

Chain Core 1.1 introduced a new pattern of identifying transaction outputs using a single unique identifier. This deprecated the 1.0-style identifier, which used a tuple of transaction ID and output position.

With Chain Core 1.2, the deprecated tuple identifier has been removed. This has two interface-level effects:

- `Transaction::Input` no longer has a `spent_output` property. Use `spent_output_id` instead.
- The `spend_account_unspent_output` transaction builder method no longer accepts the `transaction_id` and `position` parameters. Use `output_id` instead. See the [documentation](https://chain.com/docs/1.2/core/build-applications/unspent-outputs#spend-unspent-outputs) for examples.

### New features

#### Authentication and authorization

The `Chain::AuthorizationGrant` class allows you to programmatically register credentials that have well-scoped access to different parts of the Chain Core API. See the [documentation](https://chain.com/docs/1.2/core/learn-more/authentication-and-authorization) for code samples, as well as a detailed introduction to authentication and authorization in Chain Core.

#### TLS configuration

The SDK now features flexible TLS configuration, allowing you to accept server certificates outside of the public X.509 PKI, and to authenticate requests to Chain Core using client X.509 certificates.

See the [documentation](https://chain.com/docs/1.2/core/learn-more/mutual-tls-auth) for specific instructions.

## 1.2.0.rc2 (May 3, 2017)

Pre-release version.

## 1.2.0.rc1

Unpublished pre-release version.

