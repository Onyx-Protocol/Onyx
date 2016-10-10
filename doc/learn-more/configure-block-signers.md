# Create blockchain with block signers

## Introduction
The Chain Core dashboard does not yet support block signer configuration. However, you can use the Chain Core command line tools to manually configure a blockchain with block signers.

### Install command line tools
```
???
```

## Configuration
The process of configuration takes a few back and forth steps between the block generator and the block signers.

In this example, we configure a blockchain with one block signer whose signature is required.

### Signer
Note: We do not yet configure the Chain Core. We first create a block signing key in the MockHSM and a network token so the generator can submit blocks to the network API for signing.

#### Create a block signing key

```
$ corectl create-block-keypair
```

This prints out the pubkey as a hex string.

```
cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313
```

#### Create a network token
Note: `foo` is a user-supplied network token id

```
$ corectl create-token -net foo
```
This prints out the network token, which can be included as basic auth in the URL when accessing the Chain Core network API.

```
foo:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7
```

#### Send details to the block generator
This happens out of band.

```
public key: cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313

Block signer Chain Core URL with network token: https://foo:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7@<signer-url>
```

### Generator

#### Configure Chain Core
Configure Chain Core as block generator, with the additional block signer, requiring signatures from both.

* The `-s` flag includes this generator as a block signer.

```
$ corectl config-generator -s <quorum> <signer1-pubkey> <signer1-url-with-network-token>
$ corectl config-generator -s 2 cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313 https://foo:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7@<signer-url>
```
This prints out the blockchain id

```
ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c
```


#### Create a network token for the signer

```
$ corectl create-token -net signer
```
This prints out the network token,  which can be included as basic auth in the URL when accessing the Chain Core network API.

```
signer:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7
```

###### Send details to signer
This happens out of band.

```
Block generator Chain Core URL: https://<generator-url>
Network token: signer:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7
Blockchain ID: ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c
```

### Signer

#### Configure Chain Core
Configure Chain Core as block signer.

```
$ corectl config\
    -t <block generator network token> \
    -k <block signing public key> \
    <blockchain id> \
    <block genrator URL>

$ corectl config\
    -t signer:ea8b749f154a790c8a3aeb57bb2fa98968f51a991c4771fb072fc25f6563b6f7 \
    -k cce1791bf3d8bb5e506ec7159bad6a696740712197894336c027dec9fbfb9313 \
    ec95cfab939d7b8dde46e7e1dcd7cb0a7c0cea37148addd70a4a4a5aaab9616c \
    https://<generator-url>


```

Once complete, the block signer will download the initial block from the block generator, and then automatically sign new blocks as the block generator delivers them.
