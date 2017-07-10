<!---
cored command usage information.
-->

# cored Command

Command `cored` provides the Chain Core daemon and API server.

`cored` is included with all official releases of Chain Core
Developer Edition, and is run by default when starting Mac, Windows and Docker.

If you plan to make changes to the internals of Chain Core, you can find
build instructions in our [GitHub readme](https://github.com/chain/chain/blob/main/Readme.md#building-from-source);
if you want to _use_ chain core, please download one of our official releases.

### Developer Edition

Chain Core Developer Edition includes some features designed for
convenience during development and testing:

  - `reset`: allows the Chain Core data to be reset via the API or dashboard
  - `localhost_auth`: allows unauthenticated requests when accessing Chain Core from localhost
  - `mockhsm`: provides a simulated HSM interface for development
  - `http_ok`: allows unencrypted (non-TLS) HTTP requests.

The status of these features are printed on startup, and in the `cored -version`
output. These features are unsafe for use in production environments and are
disabled in [Chain Core Enterprise Edition](https://chain.com/get-in-touch/).

## Flags

* **-version**: Print version and build information

## Environment Variables

Many functions of Chain Core can be controlled by the presence of environment
variables. For example, to start a Chain Core that listens on a different port:

```
LISTEN=:9876 cored
```

### Basic Functions

The following variables control the location of Chain Core's data stores,
and its routable address.

* **CHAIN_CORE_HOME**: Path to Chain Core data directory on local file system,
defaults to `$HOME/.chaincore`.

* **DATABASE_URL**: URL of the Postgres database, defaults
to `postgres:///core?sslmode=disable`.

* **LISTEN**: Address the Chain Core server will listen on, defaults
to `:1999`. In a multi process configuration, this must be a
reachable, routable address.

### Extended Functionality

The following variables allow for more fine-grained operation control of
Chain Core, such as log rotation, rate limits, and clustering.

* **ROOT_CA_CERTS**: Path to file containing a set of PEM-encoded concatenated
root CA certificates to trust. If unset, `cored` will trust no CA certs. See the
[client TLS guide](../learn-more/mutual-tls-auth#client-authentication) for more info.

* **LOGFILE**: Path to location of base file for for Chain Core log output. Log
file can be rotated automatically based on `LOGSIZE` and `LOGCOUNT` variables.
 If unset, logs will be printed to `stdout`.

* **LOGSIZE**: Size limit of the base log file in bytes, defaults to 5MB.

* **LOGCOUNT**: Number of rotated log files to keep, defaults to 9.

* **MAXDBCONNS**: Maximum number of simultaneous connections to Postgres from
Chain Core, defaults to 10.

* **RATELIMIT_TOKEN**: Maximum number of requests-per-second
allowed with an individual access token. Requests made beyond
the limit will receive an HTTP 429 response.

    Can be stacked with **RATELIMIT_REMOTE_ADDR**.

* **RATELIMIT_REMOTE_ADDR**: Maximum number of requests-per-second
allowed fro a remote IP address. Requests made beyond
the limit will receive an HTTP 429 response.

    Can be stacked with **RATELIMIT_TOKEN**.

* **BOOTURL**: Setting this value causes the `cored` process to join an
existing Chain Core cluster if it's not already a member. If it is already
a member of a cluster, this has no effect.

    This should be the URL of any `cored` process already in the cluster,
    or a load balancer that forwards requests to any node.

## Mutual TLS

Chain Core 1.2 introduces support for mutual TLS authentication. This means both Chain Core and the client SDKs can authenticate each other using X.509 certificates and the TLS protocol. Previously, client authentication was facilitated through the use of access tokens and HTTP Basic Auth. While still supported, client access tokens are now deprecated.

For information on setting up TLS, visit the [mutual TLS guide](../learn-more/mutual-tls-auth).
