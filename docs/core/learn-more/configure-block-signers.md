# Create blockchain with block signers

## Introduction

The Chain Core dashboard does not yet support block signer configuration. However, you can use a Chain Core command line tool to manually configure a blockchain with block signers.

### Initialize Corectl

In the Chain Core Mac app, visit the `Developer` menu and select `Open Terminal`. This will initialize the `corectl` command line tool.

## Configuration

The process of configuration takes a few back and forth steps between the block proposer and the block signers.

In this example, we configure a blockchain with a Chain Core as a block proposer on one machine and another Chain Core as a block signer on another machine.

### Signer

Note: We do not yet configure the Chain Core.

On the machine that will host the block-signing core, we first create a block-signing key in the Mock HSM and a network token so the proposer can submit blocks to the signer’s network API for signing.

#### Create a block signing key

```bash
corectl create-block-keypair
```

This prints out the pubkey as a hex string.

```
cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313
```

#### Create a network token

Note: `foo` is a user-supplied network token id

```bash
corectl create-token -net foo
```

This prints out the network token, which can be included as basic auth in the URL when accessing the Chain Core network API.

```
foo:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7
```

#### Send details to the block proposer

This happens out of band.

```
public key: cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313

Block signer Chain Core URL with network token: https://foo:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7@<signer-host>:<signer-port>
```

### Proposer

#### Configure Chain Core

On the machine that will host the block proposer, configure Chain Core to require two signatures on each block: its own, plus one from the separate block signer:

```bash
corectl config-proposer -s <quorum> <signer1-pubkey> <signer1-url-with-network-token>
```

```bash
corectl config-proposer -s 2 cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313 https://foo:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7@<signer-host>:<signer-port>
```

The `-s` flag means this proposer is also a block signer. The quorum value, 2, means both signatures are required on every block.

This prints out the blockchain id

```
ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c
```

#### Create a network token for the signer

```bash
corectl create-token -net signer
```

This prints out the network token,  which can be included as basic auth in the URL when accessing the Chain Core network API.

```
signer:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7
```

###### Send details to signer

This happens out of band.

```
Block proposer Chain Core URL: https://<proposer-url>
Network token: signer:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7
Blockchain ID: ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c
```

### Signer

#### Configure Chain Core

Back on the first machine, configure Chain Core as block signer.

```bash
corectl config\
    -t <block proposer network token> \
    -k <block signing public key> \
    <blockchain id> \
    <block proposer URL>
```

```bash
corectl config\
    -t signer:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7 \
    -k cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313 \
    ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c \
    https://<proposer-host>:<proposer-port>
```

At this point it may be necessary to quit and restart the Chain Core Mac app in order to pick up the configuration changes made with `corectl`.

Once the configured cores are running, the block signer will download the initial block from the block proposer, and then automatically validate and sign new blocks as the block proposer delivers them.
