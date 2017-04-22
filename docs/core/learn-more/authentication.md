# Authentication and Authorization

## Authentication

Chain Core allows control of which entities have access to certain features of the system. There are two methods available for limiting access:

1. Access tokens using HTTP Basic Authentication
2. X.509 client certificates

These authentication objects can be created and managed via the Chain Core Dashboard, SDKs, or the [`corectl`](../reference/corectl) command line tool.

For convenience, in all desktop installations of Chain Core **access from localhost does not require authentication**.

## Authorization

There are two APIs in Chain Core: the **client API** and the **network API**.

The client API is used by the SDKs and the dashboard to communicate with Chain
Core. The network API is used by [network operators](blockchain-operators.md).

There are four policies available to grant an individual authentication method
access to one or both APIs:

* **client-readwrite**: Full access to the Client API.
* **client-readonly**: Access to read-only Client endpoints. This is a strict
subset of the `client-readwrite` policy.
* **monitoring**: Access to monitoring-specific endpoints. This is a strict
subset of the `client-readonly` policy.
* **network**: Access to the Network API.

## Setting Up

### Mac, Windows and Docker

Chain Core Developer Edition is distributed with a flag automatically permitting
all requests from localhost without configuring an authorization method.

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

$code connect-with-token ../examples/java/AccessToken.java ../examples/ruby/access_token.rb ../examples/node/accessToken.js

## Granting Access

Authorization grants map authentication methods to access policies. An
authorization grant is made up of:

1. A `guard_type`, either `access_token` or `x509`.
2. A `guard_data` object, identifying a specific token, or set of fields in an
X.509 certificate.
3. A `policy` field specifying the policy to attach.

For example, to grant access to a new party that wants to read blockchain data,
we can create a new access token, and give it the `client-readonly` policy
via an authorization grant:

$code create-read-only ../examples/java/AccessToken.java ../examples/ruby/access_token.rb ../examples/node/accessToken.js
