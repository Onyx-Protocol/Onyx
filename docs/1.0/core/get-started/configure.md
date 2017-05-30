# Configure Chain Core

## Overview

When you first launch your Chain Core and visit the dashboard at `http://HOST:1999/` (where HOST is the name of the computer where Chain Core is running), you will be presented with three options:

1. [Create a new blockchain](#create-a-new-blockchain)
2. [Connect to an existing blockchain](#connect-to-an-existing-blockchain)
3. [Connect to the Chain testnet](#connect-to-the-chain-testnet)

Choosing one of these options configures your Chain Core. You can reset it to its initial unconfigured state at any time by visiting Core Settings in the dashboard.

## Create a new blockchain

This creates a new blockchain with the Core as the block generator and single block signer. The Core's block-signing key is automatically created in the Mock HSM. The Core's URL and blockchain ID (needed by other Cores wishing to join this network) are available in Core Settings in the dashboard.

For more information, see [operating a blockchain](../learn-more/blockchain-operators.md).

## Connect to an existing blockchain

This connects to an existing blockchain whose block generator is already configured. You must supply the following information to join:

* Block generator URL
* Network access token
* Blockchain ID

Once configured, Chain Core will begin downloading blockchain data from the block generator. Once your Core is up to date with the network it will receive new blocks as they are created.

For more information, see [participating in a blockchain](../learn-more/blockchain-participants.md).

## Connect to the Chain testnet

Chain operates a public testnet for development purposes. When initializing Chain Core, choosing "Connect to Chain Testnet" will automatically connect to the Chain testnet blockchain.

### Testnet resets

**Chain Testnet is reset weekly**. Your Chain Core will automatically detect this and prompt you to reconnect to the new Chain testnet blockchain.
