# cored changelog

## 1.1.0 (February 24, 2017)

This release is a minor version update, and contains **new features** and **deprecations**. cored 1.1.0 is backward-compatible with 1.0.x SDKs, but we strongly recommend upgrading to 1.1.x SDKs as soon as possible. See SDK-level changes here:

- [Java SDK changelog](https://github.com/chain/chain/blob/main/sdk/java/CHANGELOG.md)
- [Node SDK changelog](https://github.com/chain/chain/blob/main/sdk/node/CHANGELOG.md)
- [Ruby SDK changelog](https://github.com/chain/chain/blob/main/sdk/ruby/CHANGELOG.md)

Notable changes:

* The network version has been updated to **2**. Chain Core instances on the same network must share the same network version. If you're upgrading to version 1.1.0, make sure to upgrade all Chain Cores in your blockchain network.
* Transaction outputs now have a unique `id` property.
* Transaction inputs refer to previous outputs using a new `spent_output_id` property. The existing `spent_output` property, which contains a transaction ID and position, is **deprecated**.
* Accounts now use **receivers**, a cross-core payment primitive that supersedes the Chain 1.0.x pattern of creating and paying to control programs. See the SDK changelogs for usage examples.

## 1.0.2 (December 2, 2016)<a name="1.0.2"></a>

* Resolved issue with some transactions being incorrectly marked as "not final"
  ([#160](https://github.com/chain/chain/issues/160), [#161](https://github.com/chain/chain/pulls/161))
* Added Ruby SDK documentation
* Updated included Java SDK to [1.0.1](../../sdk/java/CHANGELOG.md#1.0.1)
* MockHSM keys can now be generated automatically when creating accounts or
    assets from the Dashboard
* Bug fixes and performance improvements

## 1.0.1 (October 25, 2016)

* Updated link to Java SDK JAR file

## 1.0.0 (October 24, 2016)

* Initial release
