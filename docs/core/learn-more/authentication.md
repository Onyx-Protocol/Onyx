# Authentication and Authorization

Chain Core allows control of which entities have access to certain features of the system. There are two methods available for limiting access:

1. Access tokens using HTTP Basic Authentication
2. x509 client certificates

These authentication objects can be created and managed via the Chain Core Dashboard, SDKs, or the [`corectl`](../reference/corectl) command line tool.

For convenience, in all pre-packaged installations of Chain Core **access from localhost does not require authentication**.

## Authorization

There are two APIs in Chain Core: the **client API** and the **network API**.

The client API is used by the SDKs and the dashboard to communicate with Chain Core. The network API is used by [network operators](blockchain-operators.md).

There are four policies available to grant an individual authentication method access to one or both APIs:

* **Client read/write**: Full access to the Client API.
* **Client read-only**: Access to read-only Client endpoints. This is a strict subset of the "client read/write" policy.
* **Monitoring**: Access to monitoring-specific endpoints. This is a strict subset of the "client read-only" policy.
* **Network**: Access to the Network API.

## Setting Up

When deploying Chain Core to a non-local environment, you will not be able to access the Dashboard or APIs to create authorization grants. Therefore, you must use the `corectl` command line tool to create your first authorization grant. After that, you can use that token or certificate to create additional authorizations via the Dashboard or SDKs.

[sidenote]

Before proceeding, make sure you have `corectl` installed on your system. If it's not already present,  see [installing `corectl`](../reference/corectl#installation).

[/sidenote]

Create a new **access token** and give it the **client-readwrite** policy:

```
corectl create-token <name> client-readwrite
```

The command will return your access token:

```
<name>:<secret>
```

Anywhere that Chain Core asks for this token, it's important to provide the entire value, both name and secret, in the format returned by this command.
