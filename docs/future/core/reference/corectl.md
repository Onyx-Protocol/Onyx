# corectl Command

Command `corectl` provides miscellaneous control functions for a Chain Core.

## Installation

`corectl` is included with the Mac desktop application and Docker images by
default. For all other installations, you'll need to compile `corectl` from
source.

### Mac

Using the Mac desktop app, open the "Developer" menu, and select
"Open Terminal". A new window will be opened in your preferred terminal
application with `corectl` configured for your machine.

### Docker

When running the `chaincore/developer` Docker image, `corectl` is available in
the `/usr/bin/chain` directory of your container. You can run it with the
following command:

```
docker exec <container-name> /usr/bin/chain/corectl <arguments>
```

### From source

For Windows installations or systems compiling `cored` from source, you'll need
to manually install `corectl`.

_The instructions in this section require having the Go programming environment installed and the PATH variable correctly configured. See the [Chain Core Readme file](https://github.com/chain/chain/blob/main/Readme.md#building-from-source) for details._

Install the `corectl` command line tool:

```
$ go install chain/cmd/corectl
```

## Configuration

`corectl` requires a connection to the Chain Core server. By default, it assumes Chain Core is hosted at `http://localhost:1999`. You can configure this URL by setting the `CORE_URL` environment variable. For example:

```
CORE_URL=https://cored.example.com:9999 corectl create-token ...
```

## Commands

* [init](#init)
* [join](#join)
* [config-generator](#config-generator)
* [config](#config)
* [create-block-keypair](#create-block-keypair)
* [create-token](#create-token)
* [reset](#reset)
* [grant](#grant)
* [revoke](#revoke)
* [allow-address](#allow-address)
* [get](#get)
* [add](#add)
* [rm](#rm)
* [wait](#wait)

### `init`

Creates a new Chain Core cluster. The cored process addressed by `CORE_URL` will
be the first and only member of the new cluster. Additional members may be added
using the `allow-address` and `join` commands.

```
corectl init
```

### `join`

Connects the Chain Core process to an existing Chain Core cluster.

`join` should only be used for multiserver Chain Cores.

```
corectl join [address]
```

Argument:

* **address**: The boot address, in `host:port` format.


### `config-generator`

Configures a new core as a generator. You may optionally require multiple
additional Chain Cores as signers for this generator.

```
corectl config-generator [-k pubkey] [-w duration] [quorum] [pubkey url]...
```

Flags:

* **-k \<pubkey>**: Local pubkey for signing blocks; indicates that this core
will be a signer. If **-k** is not given, the core will be a participant (not a generator or a signer).
* **-w \<duration>**: The maximum issuance window duration for this generator (default 24h0m0s).
* **-hsm-url \<url>**: HSM url for signing blocks (MockHSM if empty).
* **-hsm-token \<access-token>**:  HSM access-token for connecting to HSM.

Arguments:

 * **[quorum]**: The number of signers _required_ to sign a block. This number
may be equal to or less than the total number of signers in the network.
 * **[pubkey url]**: Pairs of block signing pubkeys and URLs (optionally with
an authentication token embedded), one pair per additional signer requested.
Pubkeys and URLs are to be provided out-of-band by the entities running
those nodes.

### `config`

Configures the Core as a non-generator. It requires a
blockchain ID and the corresponding generator URL. Optionally, it takes
the public key of a block signing key if the Core is to be configured
as a signer.

```
corectl config [-t token] [-k pubkey] blockchain-id generator-url
```

Flags:

 * **-k \<pubkey>**: Local pubkey for signing blocks; indicates that this core
 will be a signer. If **-k** is not given, the core will be a participant (not a generator or a signer).
 * **-t \<token>**: Authentication token with access to the network API provided
by the generator.
 * **-hsm-url \<url>**: HSM url for signing blocks (MockHSM if empty).
 * **-hsm-token \<access-token>**:  HSM access-token for connecting to HSM.

Arguments:

* **blockchain-id**: ID of the generator's blockchain network.
* **generator-url**: URL of the network's block generator.

### `create-block-keypair`

Generates a new keypair in the MockHSM for block signing, with the
alias "block_key".

```
corectl create-block-keypair
```

### `create-token`

Generates a new access token with the given name
and grants it access to the named policy.

```
corectl create-token [-net] [name] [policy]
```

If no policy is given,
it grants access to policy `client-readwrite`.
This form is deprecated;
please provide a policy by name.

Flag `-net` grants access to the policy `network`
for the new token.
This flag is deprecated;
please provide a policy by name instead.

### `reset`

Resets the Chain Core configuration. All blockchain data, access tokens, and
other configuration will be deleted.

```
corectl reset
```

### `grant`

Grants access to a policy
for the described credentials.

```
corectl grant [policy] [guard]
```

Arguments:

 * **policy**: the policy to grant access to.
See [Authentication and Authorization](../learn-more/authentication-and-authorization)
for a list of policies and their meaning.
 * **guard**: indicates what authentication credentials to require.
It must take one of three forms:
   * `token=[name]` to affect an access token
   * `CN=[name]` to affect an X.509 Common Name
   * `OU=[name]` to affect an X.509 Organizational Unit

   The type of guard (before the = sign) is case-insensitive.

### `revoke`

Revokes access to a policy
for the described credentials.

```
corectl revoke [policy] [guard]
```

Arguments:

 * **policy**: the policy to revoke access from.
 * **guard**: indicates what authentication credentials to affect.
It must take one of three forms:
   * `token=[name]` to affect an access token
   * `CN=[name]` to affect an X.509 Common Name
   * `OU=[name]` to affect an X.509 Organizational Unit

   The type of guard (before the = sign) is case-insensitive.

### `allow-address`

Adds an address to the allowed members list and creates a grant with policy `internal` for this host.

`allow-address` should only be used for multiserver Chain Cores.

```
corectl allow-address [address]
```

Argument:

* **address**: The listen address, in `host:port` format, to be added to the allowed members list.


### `get`

Retrieves the current value of a configuration option.

```
corectl get [key]
```

Argument:

* **key**: The configuration option to retrieve.


### `add`

Adds a tuple to a configuration option's set of values. The configuration option must be defined as a set. The number of value arguments provided must match the configuration option's tuple length. Add will error if a conflicting tuple already exists in the set. To overwrite the existing tuple, provide the `-u` flag.

```
corectl add [key] [value]...
```

Flags:

* **-u**: Overwrite the existing tuple if it conflicts.

Arguments:

* **key**: The configuration option to modify.
* **value**: One or more values forming a single tuple to add to the set.


### `rm`

Remove a tuple from a configuration option's set of values. The configuration option must be defined as a set. The number of value arguments provided must much the configuration option's tuple length.

```
corectl rm [key] [value]...
```

Arguments:

* **key**: The configuration option to modify.
* **value**: One or more values forming a single tuple to remove from the set.


### `wait`

Blocks until the Chain Core server is available.

```
corectl wait
```
