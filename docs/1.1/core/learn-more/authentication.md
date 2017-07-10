<!---
An overview of authentication using access tokens with HTTP Basic Authentication.
-->

# Authentication

## Introduction

There are two APIs in Chain Core: the **client API** and the **network API**.

The client API is used by the SDKs and the dashboard to communicate with Chain Core. The network API is used by [network operators](blockchain-operators.md).

Each API is authenticated using access tokens with HTTP Basic Authentication.

For convenience, **when accessing from localhost, neither API requires authentication**.

## Creating access tokens

_The instructions in this section require having the Go programming environment installed and the `PATH` variable correctly configured. See [the Chain Core Readme file](https://github.com/chain/chain/blob/main/Readme.md) for details._

Both client and network access tokens are created in the dashboard. However, when deploying Chain Core to a non-local environment, you will not be able to access the dashboard, because you will not yet have a client access token. Therefore, you must use the `corectl` command line tool to create your first client access token. After that, you can use that access token to login to the dashboard and create additional access tokens.

Install the `corectl` command line tool:

```bash
go install ./cmd/corectl
```

Create a **client access token** using `corectl`:

```bash
corectl create-token <name>
```

The command will return your access token:

```
<name>:<secret>
```
