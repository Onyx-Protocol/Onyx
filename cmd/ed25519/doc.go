/*

Command ed25519 creates and manipulates ed25519 public and private keys.

Usage:

	ed25519 gen >privatekey
	ed25519 pub <privatekey >publickey
	ed25519 sign PRIVATEKEY_HEX <message >signature
	ed25519 verify [-s] PUBLICKEY_HEX SIG_HEX <message

The gen subcommand generates a new, random private key.
The pub subcommand reads a private key and produces the corresponding public key.
The sign subcommand produces a signature from a message and private key.
The verify subcommand verifies a signature with a message and a public key.

The verify subcommand prints "OK" or "BAD" to stdout unless the -s ("silent") flag is given.
The program exits with 0 when the signature is verified, nonzero when it's not.

*/
package main
