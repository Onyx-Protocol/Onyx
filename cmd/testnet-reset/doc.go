/*

Command testnet-reset is a convenient command to reset a blockchain network.
It takes no optional flags or other arguments.
It expects twelve environment variables: four each for three Chain Core deployments.
One is the proposer, the other two are signers.

	PROPOSER_URL
	PROPOSER_CLIENT_TOKEN
	PROPOSER_NETWORK_TOKEN
	PROPOSER_PUBKEY

	SIGNER1_URL
	SIGNER1_CLIENT_TOKEN
	SIGNER1_NETWORK_TOKEN
	SIGNER1_PUBKEY

	SIGNER2_URL
	SIGNER2_CLIENT_TOKEN
	SIGNER2_NETWORK_TOKEN
	SIGNER2_PUBKEY

*/
package main
