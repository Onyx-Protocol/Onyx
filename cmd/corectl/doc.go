/*

Command corectl provides miscellaneous control functions for a Chain Core.

The database connection can be configured using the DATABASE_URL environment
variable; the default is to connect to the "core" database on localhost.

The config commands initialize the schema if necessary.

Config Generator

Subcommand 'config-generator' configures a new core as a generator.
It matches the dashboard's behavior when writing the config,
but with additional functionality.

	corectl config-generator [-s] [-w duration] [quorum] [pubkey url]...

Flag -s sets this core as a signer.

Flag -w, followed by a duration string (e.g. "24h"), sets the maximum issuance window.
The default is 24 hours.

Config Participant

Subcommand 'config' configures the Core as a non-generator. It requires a
blockchain ID and the corresponding generator URL. Optionally, it takes
the public key of a block signing key if the Core is to be configured
as a signer.

	corectl config [-t token] [-k pubkey] [blockchain-id] [url]

Flag -t provides an access token to authenticate with the generator.

Flag -k causes the core to be a block signer.
Its argument is the local public key for signing blocks.
If -k is not given, the core will be a participant (not a generator or a signer).

Create Block Keypair

Subcommand 'create-block-keypair' generates a new keypair in the MockHSM for block signing,
with the alias "block_key".

    corectl create-block-keypair

Create Access Token

Subcommand 'create-token' generates a new access token with the given name.
Flag -net means to create a network token,
otherwise it will create a client token.

    corectl create-token [-net] [name]

Reset

Subcommand 'reset' resets the database so the Chain Core can be configured again.
It deletes all data.

    corectl reset

*/
package main
