# Chain Core changelog

This changelog covers updates to cored, the Chain Core server binary.

For updates to subpackages, see below:

- [Mac install](./installer/mac/CHANGELOG.md)
- [Windows installer](./installer/windows/CHANGELOG.md)
- [Java SDK](./sdk/java/CHANGELOG.md)
- [Node.js SDK](./sdk/node/CHANGELOG.md)
- [Ruby SDK](./sdk/ruby/CHANGELOG.md)

<a name="1.1.2"></a>
## 1.1.2 (March 7, 2017)

* The network version has been updated to **3**. Chain Core instances on the same network must share the same network version. If you're upgrading to version 1.1.2, make sure to upgrade all Chain Cores in your blockchain network. This version change is due to ([#648](https://github.com/chain/chain/issues/648)), which was resolved in version 1.1.1.
* Resolved issue where some transaction inputs were not correctly annotated with account information ([#668](https://github.com/chain/chain/issues/668)).
* Dashboard cosmetic changes.

<a name="1.1.1"></a>
## 1.1.1 (March 3, 2017)

* The list of root CA certificates that Chain Core uses is now configurable via the `ROOT_CA_CERTS` enviornment variable. This list is used when Chain Core acts as an TLS client, i.e. when making RPC calls to signerd or other Chain Core instances.
* Resolved issue where spent output ID was not being validated correctly ([#648](https://github.com/chain/chain/issues/648)).
* Resolved connectivity issues when bootstrapping a core from a generator snapshot ([#643](https://github.com/chain/chain/pull/643)).
* Resolved issue when connecting to other cores via HTTPS ([#674](https://github.com/chain/chain/issues/674)).

<a name="1.1.0"></a>
## 1.1.0 (February 24, 2017)

This release is a minor version update, and contains new features, deprecations, and protocol breaking changes. cored 1.1.0 is backward-compatible with 1.0.x SDKs, but we strongly recommend upgrading to 1.1.x SDKs as soon as possible. cored 1.1.0 is not backward-compatible with 1.0.X coreds due to fundamental protocol changes.

Notable changes:

* The network version has been updated to **2**. Chain Core instances on the same network must share the same network version. If you're upgrading to version 1.1.0, make sure to upgrade all Chain Cores in your blockchain network.
* Transaction outputs now have a unique `id` property.
* Transaction inputs refer to previous outputs using a new `spent_output_id` property. The existing `spent_output` property, which contains a transaction ID and position, is **deprecated**.
* Accounts now use **receivers**, a cross-core payment primitive that supersedes the Chain 1.0.x pattern of creating and paying to control programs. See the SDK changelogs for usage examples.
* The Dashboard has an improved on-boarding experience which guides new users through the basics.
* Block signing has been improved to better support HSM integration.
* Disable MockHSM and blockchain reset functions in production mode.
* Improve version string printing in cored and corectl commands.
* Bug fixes and performance improvements.

## 1.0.2 (December 2, 2016)<a name="1.0.2"></a>

* Resolved issue with some transactions being incorrectly marked as "not final"
  ([#160](https://github.com/chain/chain/issues/160), [#161](https://github.com/chain/chain/pulls/161)).
* Added Ruby SDK documentation.
* Updated included Java SDK to [1.0.1](../../sdk/java/CHANGELOG.md#1.0.1).
* MockHSM keys can now be generated automatically when creating accounts or
    assets from the Dashboard.
* Bug fixes and performance improvements.

## 1.0.1 (October 25, 2016)

* Updated link to Java SDK JAR file.

## 1.0.0 (October 24, 2016)

* Initial release.
