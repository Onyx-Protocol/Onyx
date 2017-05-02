# cored Command

Command `cored` provides the Chain Core daemon and API server.

`cored` is the primary means of running Chain Core server.

## Installation

`cored` is included with all desktop applications and Docker images by
default.

### From source

_The instructions in this section require having the Go programming environment installed and the PATH variable correctly configured. See the [Chain Core Readme file](https://github.com/chain/chain/blob/main/Readme.md#building-from-source) for details._

Install the `cored` command line tool into your Go binaries folder:

```sh
$ go install -tags 'loopback_auth plain_http reset' chain/cmd/cored
```

There are four build tags that change the behavior of the resulting binary:

  - `reset`: allows the core database to be reset through the api
  - `loopback_auth`: allows unauthenticated requests on the loopback device (localhost)
  - `no_mockhsm`: disables the MockHSM provided for development
  - `plain_http`: allows plain HTTP requests

The default build process creates a binary with three build tags enabled for a
friendlier experience. To build from source with build tags, use the following
command:


## Flags

## Environment Variables
