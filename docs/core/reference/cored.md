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

Many functions of Chain Core can be controlled by the presence of environment
variables.

### Basic Functions

The following variables control the location of Chain Core's data stores,
and its routable address.

* **CHAIN_CORE_HOME**: Path to Chain Core data directory on local file system,
defaults to `$HOME/.chaincore`.

* **DATABASE_URL**: URL of the Postgres database, defaults
to `postgres:///core?sslmode=disable`.

* **LISTEN_ADDR**: Address the Chain Core server will listen on, defaults
to `:1999`. In a multi process configuration, this must be a
reachable, routable address.

### Extended Functionality

The following variables allow for more fine-grained operation control of
Chain Core, such as log rotation, rate limits, and clustering.

* **ROOT_CA_CERTS**: Path to file containing a set of pem-encoded concatenated
root CA certificates to trust. If unset, `cored` trust no CA certs. See the
[client TLS guide](../learn-more/mutual-tls-auth#client-authn) for more info.

* **LOGFILE**: Path to location of base file for for Chain Core log output. Log
file can be rotated automatically based on `LOGSIZE` and `LOGCOUNT` variables.
 If unset, logs will be printed to `stdout`.

* **LOGSIZE**: Size limit of the base log file in bytes, defaults to 5MB.

* **LOGCOUNT**: Number of rotated log files to keep, defaults to 9.

* **MAXDBCONNS**: Maximum number of simultaneous connections to Postgres from
Chain Core, defaults to 10.

* **RATELIMIT_TOKEN**: Maximum number of requests-per-second
allowed with an individual access token. Requests made beyond
the limit will receive a `4xx` response.

    Can be stacked with **RATELIMIT_REMOTE_ADDR**.

* **RATELIMIT_REMOTE_ADDR**: Maximum number of requests-per-second
allowed fro a remote IP address. Requests made beyond
the limit will receive a `4xx` response.

    Can be stacked with **RATELIMIT_TOKEN**.

* **BOOTURL**: Setting value causes the `cored` process to join an existing
Chain Core cluster if it's not already a member. If it is already a member
of a cluster, this has no effect.

    This should be the url of any `cored` process already in the cluster,
    or a load balancer that forwards requests to any node.




## Mutual TLS

Chain Core 1.2 introduces support for mutual TLS authentication. This means both Chain Core and the client SDKs can authenticate each other using X.509 certificates and the TLS protocol. Previously, client authentication was facilitated through the use of access tokens and HTTP Basic Auth. While still supported, client access tokens are now deprecated.

For information on setting up TLS, visit the [mutual TLS guide](../learn-more/mutual-tls-auth).
