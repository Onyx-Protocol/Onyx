# Authentication and Authorization

## Authentication

There are two types of credentials that can be used to authenticate requests
to Chain Core:

1. Access tokens using HTTP Basic Authentication
2. X.509 client certificates

Credentials can be managed via the Chain Core Dashboard, SDKs, or the [`corectl`](../reference/corectl) command line tool.

For convenience, in all desktop installations of Chain Core Developer Edition **access from localhost does not require authentication**.

## Authorization

There are two APIs in Chain Core: the **client API** and the **cross-core API**.

The client API is used by the SDKs and the dashboard to communicate with Chain
Core. The cross-core API allows instances of Chain Core to communicate with each other.

There are several policies available to grant an individual credential
access to one or both APIs:

* **client-readwrite**: Full access to the Client API.
* **client-readonly**: Access to read-only Client API endpoints. This is a strict
subset of the `client-readwrite` policy.
* **monitoring**: Access to monitoring-specific endpoints. This is a strict
subset of the `client-readonly` policy.
* **crosscore**: Access to the cross-core API, including fetching blocks and submitting transactions to the [generator](blockchain-operators.md), but not including block signing. A core requires access to this policy when connecting to a generator.
* **crosscore-signblock**: Access to the cross-core API's block signing endpoint. If your blockchain network uses multiple [block signers](blockchain-operators.md), they should provide the generator with access to this policy.

## Setting Up

### Mac, Windows and Docker

For convenience, Chain Core Developer Edition permits all API requests
originating from the same host (i.e., localhost) without the need
for credentials.

If you are accessing a Chain Core DE server from an external URL, you can
follow the instructions below. Otherwise, skip ahead to
[granting access](#granting-access).

### Remote Server

When deploying Chain Core to a non-local environment, you will not be able to
access the Dashboard or APIs to create authorization grants. Therefore, you
must use the `corectl` command line tool to create your first authorization
grant. After that, you can use that token or certificate to create additional
authorizations via the Dashboard or SDKs.

[sidenote]

Before proceeding, make sure you have `corectl` installed on your system. If
it's not already present, see
[installing `corectl`](../reference/corectl#installation).

[/sidenote]

Create a new **access token** and give it the **client-readwrite** policy:

```
corectl create-token <name> client-readwrite
```

The command will return your access token:

```
<name>:<secret>
```

[sidenote]

Anywhere that Chain Core asks for a token, make sure to provide the entire
value, name and secret, in the format returned by this command.

[/sidenote]

This access token can now be used to create additional tokens and authorizations
via the Dashboard, or in the Chain SDK. To connect to a remote Chain Core
with a token from the SDK, you can pass parameters to the client constructor:

$code connect-with-token ../examples/java/AccessTokens.java ../examples/ruby/access_tokens.rb ../examples/node/accessTokens.js

### TLS Authentication

To make use of X.509 certificates for authentication, Chain Core must be
configured to trust certificates sent by the client application. See the
[mutual TLS guide](mutual-tls-auth.md) for more details.

## Granting Access

Authorization grants map credentials to access policies. An authorization grant
is made up of:

1. A `guard_type`, either `access_token` or `x509`.
2. A `guard_data` object, identifying a specific token, or set of fields in an
X.509 certificate.
3. A `policy` field specifying the policy to attach.

For example, to grant access to a new party that wants to read blockchain data,
we can create a new access token, and give it the `client-readonly` policy
via an authorization grant:

$code create-read-only ../examples/java/AccessTokens.java ../examples/ruby/access_tokens.rb ../examples/node/accessTokens.js
