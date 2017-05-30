# Setting up a blockchain with multiple block signers

This guide describes how to create a blockchain network with two Chain Cores in the consensus group, one core acting as the block generator, and the other acting as a block signer. A block must contain signatures from both parties for the block to be considered valid.

Configuring the two cores requires use of [corectl](../reference/corectl.md), a command-line configuration tool distributed with Chain Core Developer Edition.

### Signer

The block signer should start with a running, *unconfigured* instance of Chain Core. We can still use the unconfigured core to generate MockHSM keys and access credentials.

Full configuration of this core will be completed *after* the block generator is configured, but the block generator must first receive the block signer's public key and access credentials.

#### Create a block signing key

Use the [create-block-keypair](../reference/corectl.md#create-block-keypair) command to generate a new public/private keypair in the block signer's MockHSM:

```
corectl create-block-keypair
```

The output of this command is the new public key.

##### Example

```
signer-host$ corectl create-block-keypair
cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313
```

#### Create an access token for the generator

The generator will make requests to the block signer's block signing API, which is protected by the `crosscore-signblock` policy.

Use the [create-token](../reference/corectl.md#create-token) command to generate a new access token with access to the `crosscore-signblock` policy:

```
corectl create-token <token ID> crosscore-signblock
```

The output of this command is a token with access to the `crosscore-signblock` policy on the block signer. This token can be used as HTTP Basic Auth credentials when making requests to the block signer.

##### Example:

```
signer-host$ corectl create-token generatortoken crosscore-signblock
generatortoken:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7
```

#### Send details to the block generator

Out of band, the following details should be sent to the block generator:

- The block signer's key: `cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313`
- Block signer's core URL, with access token: `https://generatortoken:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7@<signer-host>:<signer-port>`

### Generator

#### Create the generator's block signing key

The block generator will also sign blocks. Use the [create-block-keypair](../reference/corectl.md#create-block-keypair) to generate a new public/private keypair in the block generator's MockHSM:

```
corectl create-block-keypair
```

The output of this command is the new public key.

##### Example

```
generator-host$ corectl create-block-keypair
45ad1f1617d4c6fb8ae5119c524c6266595f0f551c5210dd9f5892b4b39f011f
```

#### Configure Chain Core

Use the [config-generator](../reference/corectl.md#config-generator) command to configure Chain Core to require two signatures on each block, one from the generator, and one from the other block signer:

```
corectl config-generator \
    -k <generator-pubkey> \
    <quorum> \
    <signer1-pubkey> \
    <signer1-url-with-access-token>
```

The `-k` flag means this generator is also a block signer, and will provide its own signing key. The quorum value indicates how many signatures must appear in a block for the block to be considered valid. With two signers, the quorum can either be 1 or 2.

The output of this command is the blockchain ID, which is the hash of the first block.

##### Example:

```
generator-host$ corectl config-generator \
    -k 45ad1f1617d4c6fb8ae5119c524c6266595f0f551c5210dd9f5892b4b39f011f \
    2 \
    cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313 \
    https://generatortoken:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7@<signer-host>:<signer-port>
ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c
```

#### Create a network token for the signer

Like all other cores on the same network, the block signer will fetch blocks and submit transactions to the block generator's cross-core API, which is protected by the `crosscore-signblock` policy.

Use the [create-token](../reference/corectl.md#create-token) command to create an access token with access to this policy, run:

```
corectl create-token <token ID> crosscore
```

The output of this command is a token with access to the `crosscore` policy on the block generator. This token can be used as HTTP Basic Auth credentials when making requests to the block generator.

##### Example:

```
generator-host$ corectl create-token signertoken crosscore
signertoken:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7
```

#### Send details to signer

Out of band, the following information should be sent to the block signer:

- Block generator's core URL: `https://<generator-url>`
- Generator access token: `signertoken:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7`
- Blockchain ID: `ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c`

### Signer

#### Configure Chain Core

Use the [config-generator](../reference/corectl.md#config-generator) command to configure the block signer's core to perform block signing:

```
corectl config \
    -t <block generator access token> \
    -k <block signing public key> \
    <blockchain id> \
    <block generator URL>
```

Once the configured cores are running, the block signer will download the initial block from the block generator, and then automatically validate and sign new blocks as the block generator delivers them.

##### Example:

```
signer-host$ corectl config \
    -t signertoken:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7 \
    -k cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313 \
    ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c \
    https://<generator-host>:<generator-port>
```
