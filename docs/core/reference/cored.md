# cored Command

Command `cored` provides the Chain Core daemon and API server.

## Installation

`cored` is included with all desktop installations of Chain Core
Developer Edition, and is run by default when starting Mac, Windows and Docker.

### From source

_The instructions in this section require having the Go programming environment installed and the PATH variable correctly configured. See the [Chain Core Readme file](https://github.com/chain/chain/blob/main/Readme.md#building-from-source) for details._

There are four build tags that change the behavior of the resulting binary:

  - `reset`: allows the core database to be reset through the api
  - `loopback_auth`: allows unauthenticated requests on the loopback device (localhost)
  - `no_mockhsm`: disables the MockHSM provided for development
  - `plain_http`: allows plain HTTP requests

The default build process creates a binary with three build tags enabled for a
friendlier experience. To build a binary from source with a set of build tags,
use the following command:

```sh
go build -tags 'loopback_auth plain_http reset' chain/cmd/cored
```

## Flags

* **-version**: Print version and build information

## Environment Variables

* **ROOT_CA_CERTS**:
* **LISTEN_ADDR**
* **DATABASE_URL**
* **SPLUNKADDR**
* **LOGFILE**
* **LOGSIZE**
* **LOGCOUNT**
* **LOG_QUERIES**
* **MAXDBCONNS**
* **RATELIMIT_TOKEN**
* **RATELIMIT_REMOTE_ADDR**
* **LOGSIZE**
* **LOGSIZE**
* **INDEX_TRANSACTIONS**
* **INDEX_TRANSACTIONS**
* **CHAIN_CORE_HOME**

## Server TLS

Chain Core 1.2 introduces support for mutual TLS authentication. This means both Chain Core and the client SDKs can authenticate each other using X.509 certificates and the TLS protocol. Previously, client authentication was facilitated through the use of access tokens and HTTP Basic Auth. While still supported, client access tokens are now deprecated.

For information on setting up TLS, visit the [mutual TLS guide](../learn-more/mutual-tls-auth).
