/*

Command corectl provides miscellaneous control functions for a Chain Core.

The database connection can be configured using the DATABASE_URL environment
variable; the default is to connect to the "core" database on localhost.

    corectl config-generator

config-generator configures a new core as a generator. It matches the dashboard's
behavior when writing the config.

    corectl create-block-keypair

create-block-keypair generates a new keypair in the MockHSM for block signing,
with the alias "block_key".

    corectl config <blockchainID> <generatorURL> [block-signing-key]

corectl config configures the Core as a non-generator. It requires a
blockchain ID and the corresponding generator URL. Optionally, it takes
the public key of a block signing key if the Core is to be configured
as a signer.

*/
package main
