<!---
This changelog covers updates to cored, the Chain Core server binary.
-->

# Changelog

This changelog covers updates to cored, the Chain Core server binary.

For updates to subpackages, see below:

- [Mac installer](https://github.com/chain/chain/blob/1.2-stable/desktop/mac/CHANGELOG.md)
- [Windows installer](https://github.com/chain/chain/blob/1.2-stable/desktop/windows/CHANGELOG.md)
- [Java SDK](https://github.com/chain/chain/blob/1.2-stable/sdk/java/CHANGELOG.md)
- [Node.js SDK](https://github.com/chain/chain/blob/1.2-stable/sdk/node/CHANGELOG.md)
- [Ruby SDK](https://github.com/chain/chain/blob/1.2-stable/sdk/ruby/CHANGELOG.md)

<a name="1.2.1"></a>
## 1.2.1 (June 13, 2017)

### Major changes

TLS version 1.2 is now required for all HTTPS connections ([#1314](https://github.com/chain/chain/pull/1314)).

For Java 7 applications, TLS 1.2 must be explicitly specified in the SSLContext
object:

```
SSLContext context = SSLContext.getInstance("TLSv1.2");
context.init(null, null, null);
SSLContext.setDefault(context);
```

Clients using Node and Ruby will depend on the system-supplied OpenSSL,
which must be 1.0.1c or later.

NOTE: The system provided Ruby 2.0.0 on macOS Sierra and earlier does not
support TLS 1.2.

Other fixes:

* Performance improvements when submitting transactions containing large
numbers of assets and issuances ([#1221](https://github.com/chain/chain/pull/1221)).
* Improved checks for invalidating expired transactions ([1226](https://github.com/chain/chain/pull/1226))
* Resolved multiple unexpected crashes ([#1283](https://github.com/chain/chain/pull/1283), [#1310](https://github.com/chain/chain/pull/1310), [#1335](https://github.com/chain/chain/pull/1335))

<a name="1.2.0"></a>
## 1.2.0 (May 12, 2017)

### Breaking changes

#### Local filesystem requirement

Chain Core now requires the use of the local filesystem for persistent data. As such, platforms with ephemeral filesystems are no longer compatible with Chain Core. In particular, **deploys to Heroku are no longer recommended nor supported**. While we may investigate future workarounds for Developer Edition, you should assume that Chain Core must be deployed onto a host with a stable filesystem.

**Docker users** should take care to mount a volume for the Chain Core data directory, or else the server state will not persist between runs of the container:

```
$ docker run -p 1999:1999 \
    -v /path/to/store/datadir:/root/.chaincore \
    -v /path/to/store/postgres:/var/lib/postgresql/data \
    -v /path/to/store/logs:/var/log/chain \
    --name chaincore \
    chaincore/developer
```

#### Receivers instead of control programs

Control programs have been removed as first-class objects in the SDK, and have been superseded by the more flexible receiver interface. The programming workflow for receivers is very similar to that of control programs. You can create receivers under accounts, and use receivers in transactions using the pay-to-receiver action.

To learn more, see the [Receivers example](../build-applications/control-programs#receivers) in the Control Programs guide.

#### `corectl` requires a running server

The `corectl` utility is now a client of the Chain Core API. Before running `corectl`, make sure you have a running instance of Chain Core available. See a full list of subcommands in the [`corectl` guide](corectl.md).

### Deprecations

#### "Network" API

The Network API, which facilitates cross-core communication (such as between the generator core and another core), has been renamed to "Cross-core API". This name change has been reflected in the documentation and in the SDKs.

#### Access token `type` property

The `type` property of access tokens has been superseded by a more flexible authorization scheme using the new <a href="#1.2.0-authorization-grants">authorization grant</a> interface.

You can still create access tokens of type `client` or `network`, but they are implemented as tokens with `client-readwrite` and `crosscore` authorization grants, respectively. If you're using the access token API to programmatically generate tokens, you should migrate to a workflow that first creates tokens and then applies policy grants to them.

### New features

<a name="1.2.0-tls-client-authentication"></a>
#### TLS client authentication

You can now use X.509 client certificates instead of access tokens to provide authentication for requests to Chain Core. To do so, run Chain Core with an environment specifying the path to a file containing a list of CA certs corresponding to the client certificate issuers:

```
$ ROOT_CA_CERTS=/path/to/certs.pem cored
```

To authenticate your applications using client certificates, see the [Authentication and Authorization guide](../learn-more/authentication-and-authorization.md#tls-authentication).

<a name="1.2.0-authorization-grants"></a>
#### Authorization grants

Authorization grants allow your applications to use credentials that have limited access to the Chain Core API. For example, you may wish to grant read-only access to a particular client application, or monitoring-only access for uptime reporting.

To learn more, see the [Authentication and Authorization guide](../learn-more/authentication-and-authorization.md#granting-access).

#### Tag updating

Account and asset tags can now be updated via the dashboard and SDKs. To learn more, see examples for [accounts](../build-applications/accounts.md#update-tags-on-existing-accounts) and [assets](../build-applications/assets.md#update-tags-on-existing-assets).

<a name="1.2rc2"></a>
## 1.2rc2 (May 4, 2017)

Pre-release version.

## 1.2rc1

Unpublished pre-release version.
