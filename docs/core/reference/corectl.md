# corectl Command

Command `corectl` provides miscellaneous control functions for a Chain Core.

The database connection can be configured using the DATABASE_URL environment
variable; the default is to connect to the "core" database on localhost.

The config commands initialize the schema if necessary.

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

## Commands

* [migrate](#migrate)
* [config-generator](#config-generator)
* [config](#config)
* [create-block-keypair](#create-block-keypair)
* [create-token](#create-token)
* [reset](#reset)
* [grant](#grant)
* [revoke](#revoke)

### `migrate`

Applies any pending database migrations, ensuring the database has an
up-to-date schema.

```
corectl migrate [-status]
```

Flags:

* **-status**: Print all migrations and their status.

### `config-generator`

Configures a new core as a generator. You may optionally require multiple
additional Chain Cores as signers for this generator.

```
corectl config-generator [-s] [-w duration] [quorum] [pubkey url]...
```

Flags:

* **-k \<pubkey>**: Local pubkey for signing blocks; indicates that his core
will be a signer. If **-k** is not given, the core will be a participant (not a generator or a signer).
 * **-w \<duration>**: The maximum issuance window duration for this generator (default 24h0m0s).
 * **-hsm-url \<url>**: HSM url for signing blocks (Mock HSM if empty).
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

 * **-k \<pubkey>**: Local pubkey for signing blocks; indicates that his core
 will be a signer. If **-k** is not given, the core will be a participant (not a generator or a signer).
 * **-t \<token>**: Authentication token with access to the network API provided
by the generator.
 * **-hsm-url \<url>**: HSM url for signing blocks (Mock HSM if empty).
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

Resets the all Chain Core configuration. All blockchain data, access tokens, and
other configuration will be deleted.

```
corectl reset
```

### `grant`

Command grant grants access to a policy
for the described credentials.

```
corectl grant [-cn name] [-ou name] [-t token] [policy]
```

Flags:

 * **-cn \<name>**: Common Name;
this grants access to any valid X.509 certificate
with the given Common Name.

 * **-ou \<name>**: Organizational Unit;
this grants access to any valid X.509 certificate
with the given Organizational Unit.

 * **-t \<token>**: Token;
this grants access to the specified access token.

If multiple flags are given,
they will each take effect individually.
That is, three separate grants will occur,
not one grant with three restrictions on it.

Arguments:

 * **policy**: the policy to grant access to.
See [Authentication and Authorization](../learn-more/authentication)
for a list of policies and their meaning.

### `revoke`

Command revoke revokes access to a policy
for the described credentials.

```
corectl revoke [-cn name] [-ou name] [-t token] [policy]
```

Flags:

 * **-cn \<name>**: Common Name;
this revokes access for any valid X.509 certificate
with the given Common Name.

 * **-ou \<name>**: Organizational Unit;
this revokes access for any valid X.509 certificate
with the given Organizational Unit.

 * **-t \<token>**: Token;
this revokes access for the specified access token.

If multiple flags are given,
they will each take effect individually.

Arguments:

 * **policy**: the policy to revoke access from.
