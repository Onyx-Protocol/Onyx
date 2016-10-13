# Configure Chain Core

## Overview

When you first launch your Chain Core and visit the dashboard, you will be presented with three options:

1. Create a new blockchain
2. Connect to an existing blockchain
3. Connect to the Chain testnet

Once you configure your Chain Core, you can reset it at any time in the dashboard by visiting Core Settings.

## Create a new blockchain

This creates a new blockchain with the Core as the block generator and single block signer. The block generator key is automatically created in the MockHSM. For more information, see [operating a blockchain](/doc/learn-more/blockchain-operators).

## Connect to an existing blockchain

This connects to an existing blockchain by providing the following:

* Block generator URL
* Blockchain ID
* Network token

Once configured, Chain Core will begin downloading all existing blocks from the block generator, in order of creation. Once all blocks are downloaded, Chain Core will open a persistent connection with the block generator to receive new blocks as they are created.

For more information, see [participating in a blockchain](/doc/learn-more/blockchain-participants).

## Connect to the Chain testnet

Chain operates a public testnet for development purposes. When initializing Chain Core, choosing "Connect to Chain Testnet" will automatically connect to the Chain testnet blockchain.

### Testnet resets

**Chain Testnet is reset every two weeks**. Chain Core will automatically detect a reset and prompt you to reset your Chain Core and connect to the new Chain testnet blockchain.
