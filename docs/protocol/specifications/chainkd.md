# Chain Key Derivation

* [Introduction](#introduction)
* [Definitions](#definitions)
* [Algorithms](#algorithms)
  * [Generate root key](#generate-root-key)
  * [Generate extended public key](#generate-extended-public-key)
  * [Derive hardened extended private key](#derive-hardened-extended-private-key)
  * [Derive non-hardened extended private key](#derive-non-hardened-extended-private-key)
  * [Derive non-hardened extended public key](#derive-non-hardened-extended-public-key)
  * [Extract public key](#extract-public-key)
  * [Extract signing key](#extract-signing-key)
  * [Prune root scalar](#prune-root-scalar)
  * [Prune intermediate scalar](#prune-intermediate-scalar)
  * [Encode public key](#encode-public-key)
* [Design rationale](#design-rationale)
* [Security](#security)
* [Test vectors](#test-vectors)
* [References](#references)

## Introduction

**ChainKD** is a deterministic key derivation scheme. Generated keys are compatible with the EdDSA signature scheme and with Diffie-Hellman key exchange.

Features:

1. Fully deterministic and allows producing complex hierarchies of keys from a single high-entropy seed.
2. Can derive private keys from extended private keys using “hardened derivation.”
3. Can derive public keys independently from private keys using “non-hardened derivation.”
4. Hardened and non-hardened public keys and signatures are compatible with the [EdDSA](https://tools.ietf.org/html/rfc8032) specification. Signing keys are 64-byte strings, the format used in the [NaCl](https://nacl.cr.yp.to/sign.html) function `crypto_sign`.
5. Derivation with variable-length string selectors instead of fixed-length 31-bit integer indices.
6. Short 64-byte extended public and private keys without special encoding.
7. No metadata: an extended key carries only an additional 32-byte salt that allows sharing a public key without revealing its child public keys.
8. Privacy: extended key does not reveal neither its place in the hierarchy, nor the manner of derivation (hardened or non-hardened).

Limitations:

1. Depth of non-hardened derivation is limited to 2<sup>20</sup> (more than a million levels).
2. Number of distinct root keys or hardened public keys is 2<sup>250</sup>, half of the keyspace allowed in EdDSA.
3. Number of distinct non-hardened public keys is 2<sup>230</sup>.


## Definitions

**Selector** is a variable-length byte string indexing a child key during key derivation.

**Secret scalar** is 32-byte string representing a 256-bit integer using little-endian convention.

**Extended private key** (aka “xprv”) is a 64-byte string representing a key that can be used for deriving *child extended private and public keys*.

**Extended public key** (aka “xpub”) is a 64-byte string representing a key that can be used for deriving *child extended public keys*.

**Derivation key** (aka “dk”) is the second half (32-byte string) of the extended private or public key that enables derivation of the child keys or proving linkability of the keys.

**Signing key** (aka “sk”) is a 64-byte string (consisting of a 32-byte scalar and a 32-byte “prefix”) representing a raw signing key used for creating EdDSA signatures. This is the format used by the `crypto_sign` function in the [NaCl](https://nacl.cr.yp.to/sign.html) library.

**Private key** is the first half of the _signing key_ representing a raw signing scalar usable for ECDH and generating a _public key_.

**Public key** (aka “pk”) is a 32-byte string encoding an elliptic curve point on Ed25519 as defined in EdDSA ([RFC 8032](https://tools.ietf.org/html/rfc8032)).


## Algorithms

### Generate root key

**Input:** `seed`, a seed byte sequence (variable-length, should contain at least 256 bits of entropy).

**Output:** `xprv`, a root extended private key.

1. Compute the 64-byte string `xprv = HMAC-SHA512(key: "Root", data: seed)`.
2. [Prune the root scalar](#prune-root-scalar) defined by the first 32 bytes of `xprv`.
3. Return `xprv`.


### Generate extended public key

**Input:** `xprv`, an extended private key.

**Output:** `xpub`, an extended public key.

1. Split `xprv` into two halves: a 32-byte scalar `s` and a 32-byte derivation key `dk`.
2. Perform a fixed-base scalar multiplication `P = s·B` where `B` is a base point of Ed25519.
3. [Encode](#encode-public-key) point `P` as `pubkey`.
4. Return the extended public key `xpub = pubkey || dk` (64 bytes).


### Derive hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two halves: 32-byte scalar `s` and 32-byte derivation key `dk`.
2. Compute `xprv’ = HMAC-SHA512(key: dk, data: "H" || s || selector)`.
3. [Prune the root scalar](#prune-root-scalar) defined by the first 32 bytes of `xprv’`.
4. Return `xprv’`.


### Derive non-hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length derivation index.

**Output:** `xprv’`, the derived extended public key.

1. [Generate extended public key](#generate-extended-public-key) `xpub` for the given `xprv`.
2. Split `xpub` into two halves: a 32-byte pubkey `P` and a 32-byte derivation key `dk`.
3. Compute `F = HMAC-SHA512(key: dk, data: "N" || P || selector)`.
4. Split `F` into two halves: a 32-byte `fbuffer` and a 32-byte `dk’`.
5. [Prune intermediate scalar](#prune-intermediate-scalar) `fbuffer` and interpret it as a scalar `f` using little-endian notation.
6. Split `xprv` in two halves: a 32-byte scalar `s` and a 32-byte `dk`.
7. Compute derived secret scalar `s’ = s + f` (without reducing the result modulo the subgroup order).
8. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
9. Return `xprv’ = privkey’ || dk’`.


### Derive non-hardened extended public key

**Inputs:**

1. `xpub`, an extended public key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xpub’`, the derived extended public key.

1. Split `xpub` into two halves: a 32-byte pubkey `P` and a 32-byte derivation key `dk`.
2. Compute `F = HMAC-SHA512(key: dk, data: "N" || P || selector)`.
3. Split `F` into two halves: a 32-byte `fbuffer` and a 32-byte `dk’`.
4. [Prune intermediate scalar](#prune-intermediate-scalar) `fbuffer` and interpret it as a scalar `f` using little-endian notation.
5. Perform a fixed-base scalar multiplication `F = f·B` where `B` is a base point of Ed25519.
6. Perform point addition `P’ = P + F`.
7. [Encode](#encode-public-key) point `P’` as `pubkey’`.
8. Return `xpub’ = pubkey’ || dk’`.


### Extract public key

**Input:** `xpub`, an extended public key.

**Output:** `pubkey`, a 32-byte [EdDSA](https://tools.ietf.org/html/rfc8032) public key.

1. Return the first 32 bytes of `xpub` as encoded `pubkey` suitable for ECDH key exchange or EdDSA signatures.

The resulting 32-byte public key can be used to verify an EdDSA signature created with the corresponding [EdDSA signing key](#extract-signing-key).


### Extract signing key

**Input:** `xprv`, an extended private key.

**Output:** `sk`, a 64-byte [EdDSA](https://tools.ietf.org/html/rfc8032) signing key.

1. Compute the hash `exthash = HMAC-SHA512(key: "Expand", data: xprv)`.
2. Extract `privkey` as the first 32 bytes of `xprv`.
3. Extract `ext` as the last 32 bytes of `exthash`.
4. Return the 64-byte signing key `sk = privkey || ext`.

The resulting 64-byte signing key can be used to create an EdDSA signature verifiable by the corresponding [EdDSA public key](#extract-public-key).

### Prune root scalar

**Input:** `s`, a 32-byte string

**Output:** `s’`, a 32-byte pruned scalar

1. Clear the lowest 3 bits of the first byte.
2. Clear the highest bit of the last byte.
3. Set the second highest bit of the last byte.
4. Clear the third highest bit of the last byte.

Example:

        s[0]  &= 248
        s[31] &= 31
        s[31] |= 64


### Prune intermediate scalar

**Input:** `f`, a 32-byte string

**Output:** `f’`, a 32-byte pruned scalar

1. Clear the lowest 3 bits of the first byte.
2. Clear the highest 23 bits of the last 3 bytes.

Example:

	    f[0]  &= 248
	    f[29] &= 1
	    f[30]  = 0
	    f[31]  = 0


### Encode public key

**Input:** `P`, a Ed25519 curve point.

**Output:** `pubkey`, a 32-byte string representing a point on Ed25519 curve.

1. First encode the y coordinate (in the range 0 <= y < p) as a little-endian string of 32 bytes. The most significant bit of the final byte is always zero.
2. To form the encoding of the point `P`, copy the least significant bit of the x coordinate to the most significant bit of the final byte.
3. Return the resulting 32-byte string as `pubkey`.


## Design rationale

### Names

**ChainKD** stands for “Chain Key Derivation.”

**Xpub** and **xprv** are terms adopted from the [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki)
scheme where public and private keys are _extended_ with additional entropy (“derivation key”).

**Derivation key** or **dk** is an additional 32-byte code stored within _xpub_ and _xprv_
that allows deriving child keys and proving linkage between any pair of child keys.
In BIP32 that code is called the “chain code.” We chose not to reuse the term of BIP32 in order
to make it more explicit that the derivation key is _semi-private_.

### Deriving hardened keys from non-hardened ones

The derivation method only affects the relationship between the key and its parent.
It does not affect how other keys are derived from that key.
Note that secrecy of all derived private keys (both hardened and non-hardened, 
at all levels) from a non-hardened key depends on keeping either the parent 
extended public key secret, or all non-hardened sibling keys secret.

### Variable-length selectors instead of integers

[BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki) uses
31-bit indices for derivation. ChainKD generalizes indices to arbitrary-length 
binary strings called “selectors.”

In our experience index-based derivation is not always convenient and can be 
extended to longer selectors only through additional derivation levels which 
is less efficient (e.g. 128-bit selectors would require 5 scalar multiplications 
in BIP32). However, users are free to use integer selectors by simply encoding 
them as 32-bit or 64-bit integers and passing to ChainKD. If you need to mix 
integer- and string-based indexing, you could prepend a type byte or use a standard 
encoding such as [Protocol Buffers](https://developers.google.com/protocol-buffers/) 
or [JSON](http://www.json.org).

### Torsion point safety and scalar compatibility

The Edwards curve 25519 allows small-subgroup attacks when secret scalars are used for 
Diffie-Hellman key exchange. To prevent leaking a few bits of the scalar, the Curve25519
protocol and later EdDSA protocol require that the secret scalar is pre-multiplied by 8.
That way, when the scalar is multiplied by a torsion point, the result is always the point at infinity
which leaks no information about the scalar.

An alternative mechanism for providing safety against small-subgroup attacks is using
a _torsion-safe representative_ of a scalar: a transformation `t(s)` that keeps the
public key unmodified (`t(s)·B == s·B`) but that always yields the point at infinity when
multiplied by a torsion point (`t(s)·T == O`). In other words, `t(s) == s mod l` 
and `t(s) == 0 mod 8`.

That transformation was [proposed](https://moderncrypto.org/mail-archive/curves/2017/000866.html) 
by Henry de Valence, Ian Goldberg, George Kadianakis and Isis Lovecruft on the “Curves” mailing list
together with an efficient time-constant implementation based on a precomputed table of 8 scalars.

The torsion-safe representative is indispensable when the keys are being blinded via multiplication 
(such as in the key blinding scheme in [Tor proposal 224](https://gitweb.torproject.org/torspec.git/tree/proposals/224-rend-spec-ng.txt#n1979)).

Unfortunately, torsion-safe representation and blinding by multiplication affect
the lower and higher bits that are statically defined by Curve25519 and EdDSA: 
the 3 lower bits must be zero and the highest bits must be 1 and 0 (assuming the scalar
is a 32-byte little-endian integer). Software that implements the Montgomery ladder
or scalar multiplication may make assumptions about the value of these bits,
placing constraints on the schemes doing linear operation on the keys.

As a result, to maintain compatibility with EdDSA requirements, ChainKD uses
the trick described in [BIP32-Ed25519](https://drive.google.com/open?id=0ByMtMw2hul0EMFJuNnZORDR2NDA):
child keys are blinded via addition of a base point multiple instead of 
multiplying the parent key (like in BIP32), and the magnitute of the child scalar
is computed to be a multiple of the cofactor (8) and several bits smaller in order 
to not affect the two highest bits even after deriving many levels deep.

As a result, for all derived keys, the low 3 bits remain zero and the high 2 bits are
set to 1 and 0 as required by EdDSA, but at the cost of a hard limit 
on the derivation depth and a slightly increased (yet negligible) chance
of child key collisions.

Using derivation via addition of a blinding factor (`s’ = s + f` instead of `s’ = s·f`)
also provides better performance: scalars are faster to add than multiply, and
multiplying the factor `f` by a fixed base point can be made significantly faster
than scalar multiplication by an arbitrary point.

### Signing compatibility

EdDSA defines the private key as a 32-byte random string. The signing procedure then expands 
that private key into a 64-byte hash, where the first half is pruned to be a valid scalar, and the 
second half is used as a “prefix” to be mixed with the message to generate a secret nonce.

Unfortunately, this definition of a private key is not compatible with linear operations
required for _non-hardened_ derivation. This means that no key derivation scheme that
needs independent key derivation for secret and public keys is able to produce private keys
fully compatible with the EdDSA signing algorithm.

However, the [NaCl library](http://nacl.cr.yp.to/sign.html) happens to decouple most of the
signing logic from the key representation and uses that 64-byte hash as its “secret key”
representation for the `sign` function that we call an “expanded private key.”

ChainKD and the [BIP32-Ed25519](https://drive.google.com/open?id=0ByMtMw2hul0EMFJuNnZORDR2NDA)
proposal use that 64-byte representation to maximize compatibility with existing EdDSA implementations.
Our proposal differs slightly from BIP32-Ed25519 in that the private key does not carry additional 32 bytes
of secret entropy just to derive the “prefix” half of the expanded key. Instead, ChainKD
derives the second half of the 64-byte expanded key from the secret scalar itself, reusing its entropy
and allowing the xprv to be of the same size as the xpub — only 64 bytes (scalar + derivation key).


## Security

### Capabilities

Knowledge of the seed or the root extended private key:

1. Allows deriving a hardened extended private key;
2. Allows deriving a non-hardened extended private key;
3. Allows signing messages with the root key.

Knowledge of an extended private key:

1. Allows deriving a hardened extended private key;
2. Allows deriving a non-hardened extended private key;
3. Allows signing messages with that key.

Knowledge of an extended public key:

1. Allows deriving non-hardened public keys;
2. Does not allow determining if it is derived in a hardened or non-hardened way;
3. Does not allow determining if another extended public key is a sibling of the key;
4. Does not allow signing;
5. Does not allow deriving private keys;
6. Does not allow deriving hardened public keys.

Knowledge of a parent extended public key and one of its non-hardened derived extended private keys:

1. Allows extracting the parent private key: `s = (s’ - f) mod L` where `f` is derived from the parent `xpub` and `s’` is extracted from the child `xprv’`.

### Root key security

We set 6 bits in the secret 256 bits of a 512-bit root extended key. Therefore, cracking the root key requires on the order of 2<sup>250</sup> attempts by brute-force.

### Hardened derivation security

Private keys derived using hardened derivation have 6 bits set, just like the root key. Therefore, cracking an extended private key requires on the order of 2<sup>250</sup> attempts by brute-force.

### Non-hardened derivation security

Non-hardened derivation consist of adding scalars less that 2<sup>230</sup> (multiplied by 8) to a root private key. The resulting keys have the entropy of the root key (250 bits), but the number of possible public keys is reduced to 2<sup>230</sup> to allow a large number of derivation levels. This means collisions of public keys are expected after deriving 2<sup>115</sup> keys. We note that the increased probability of collisions does not reduce the security of EdDSA signatures or ECDH key exchange; it only marginally reduces unlinkability safety in privacy schemes based on one-time keys.

### Secret scalar compatibility

EdDSA requires specific values for 5 bits of the secret scalar: lower 3 bits must be zero, higher 2 bits must be 1 and 0.

By setting the high three bits of a root key (or hardened private key) to `010` and the low three bits to `000`, that key has form `r = 2^254 + 8·k`, where the maximum value of `k` is `2^250 - 1`. Each non-hardened derived scalar `f` is generated from 230 bits and has maximum value `2^230 - 1`. Therefore a key at level `i` has maximum value `2^254 + 2^253 - 8 + i·8·(2^230 - 1)`. Since the maximum `i` equals `2^20`, the maximum value of any key is `2^255 - 2^23 - 8`. As a result, all deriveable keys are less than `2^255`, larger than `2^254` and divisible by 8 as required by EdDSA.

The depth limit is reset at each level where hardened derivation is used.

### Nonce entropy

EdDSA derives a 64-byte signing key from 256 bits of entropy. In ChainKD the extended private key carries the secret scalar as-is and 32 bytes of _derivation key_. The 64-byte signing key as required by EdDSA consists of a secret scalar (unmodified) and an additional 32 bytes of _prefix_ used to generate the nonce for the signature.

In ChainKD that prefix is derived non-linearly from the extended private key, having the entropy of the secret scalar (250 bits). The prefix is not derived in parallel with the secret scalar, but from it, making the construction similar to the one in [RFC6979](https://tools.ietf.org/html/rfc6979) where the nonce is also computed from a secret scalar and a message using an HKDF construction.



## Test vectors

All values use hexadecimal encoding.

### ChainKD test vector 1

    Root:
        seed:     010203
        xprv:     50f8c532ce6f088de65c2c1fbc27b491509373fab356eba300dfa7cc587b07483bc9e0d93228549c6888d3f68ad664b92c38f5ea8ca07181c1410949c02d3146
        xpub:     e11f321ffef364d01c2df2389e61091b15dab2e8eee87cb4c053fa65ed2812993bc9e0d93228549c6888d3f68ad664b92c38f5ea8ca07181c1410949c02d3146

    Root/010203(H):
        selector: 010203
        xprv:     6023c8e7633a9353a59bd930ea6dc397e400b1088b86b4a15d8de8567554df5574274bc1a0bd93b4494cb68e45c5ec5aefc1eed4d0c3bfd53b0b4e679ce52028
        xpub:     eabebab4184c63f8df07efe31fb588a0ae222318087458b4936bf0b0feab015074274bc1a0bd93b4494cb68e45c5ec5aefc1eed4d0c3bfd53b0b4e679ce52028

    Root/010203(N):
        selector: 010203
        xprv:     705afd25a0e242b7333105d77cbb0ec15e667154916bbed5084c355dba7b0748b0faca523928f42e685ee6deb0cb3d41a09617783c87e9a161a04f2207ad4d2f
        xpub:     c0bbd87142e7bf90abfbb3d0cccc210c6d7eb3f912c35f205302c86ae9ef6eefb0faca523928f42e685ee6deb0cb3d41a09617783c87e9a161a04f2207ad4d2f

    Root/010203(H)/""(N):
        selector: (empty string)
        xprv:     7023f9877813348ca8e67b29d551baf98a43cfb76cdff538f3ff97074a55df5560e3aa7fb600f61a84317a981dc9d1f7e8df2e8a3f8b544a21d2404e0b4e480a
        xpub:     4e44c9ab8a45b9d1c3daab5c09d73b01209220ea704808f04feaa3614c7c7ba760e3aa7fb600f61a84317a981dc9d1f7e8df2e8a3f8b544a21d2404e0b4e480a

    Root/010203(N)/""(H):
        selector: (empty string)
        xprv:     90b60b007e866dacc4b1f844089a805ffd78a295f5b0544034116ace354c58523410b1e6a3c557ca90c322f6ff4b5e547242965eaed8c34767765f0e05ed0e4f
        xpub:     ca97ec34ef30aa08ebd19b9848b11ebadf9c0ad3a0be6b11d33d9558573aca633410b1e6a3c557ca90c322f6ff4b5e547242965eaed8c34767765f0e05ed0e4f

    Root/010203(N)/""(N):
        selector: (empty string)
        xprv:     d81ba3ab554a7d09bfd8bda5089363399b7f4b19d4f1806ca0c35feabf7b074856648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b
        xpub:     28279bcb06aee9e5c0302f4e1db879ac7f5444ec07266a736dd571c21961427b56648f55e21bec3aa5df0bce0236aea88a4cc5c395c896df63676f095154bb7b


### ChainKD test vector 2

    Root:
        seed:     fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542
        xprv:     0031615bdf7906a19360f08029354d12eaaedc9046806aefd672e3b93b024e495a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798
        xpub:     f153ef65bbfaec3c8fd4fceb0510529048094093cf7c14970013282973e117545a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798

    Root/0(N):
        selector: 00
        xprv:     883e65e6e86499bdd170c14d67e62359dd020dd63056a75ff75983a682024e49e8cc52d8e74c5dfd75b0b326c8c97ca7397b7f954ad0b655b8848bfac666f09f
        xpub:     f48b7e641d119b8ddeaf97aca104ee6e6a780ab550d40534005443550ef7e7d8e8cc52d8e74c5dfd75b0b326c8c97ca7397b7f954ad0b655b8848bfac666f09f

    Root/0(N)/2147483647(H):
        selector: ffffff7f
        xprv:     5048fa4498bf65e2b10d26e6c99cc43556ecfebf8b9fddf8bd2150ba29d63154044ef557a3aa4cb6ae8b61e87cb977a929bc4a170e4faafc2661231f5f3f78e8
        xpub:     a8555c5ee5054ad03c6c6661968d66768fa081103bf576ea63a26c00ca7eab69044ef557a3aa4cb6ae8b61e87cb977a929bc4a170e4faafc2661231f5f3f78e8

    Root/0(N)/2147483647(H)/1(N):
        selector: 01
        xprv:     480f6aa25f7c9f4a569896f06614303a697f00ee8d240c6277605d44e0d63154174c386ad6ae01e54acd7bb422243c6055058f4231e250050134283a76de8eff
        xpub:     7385ab0b06eacc226c8035bab1ff9bc6972c7700d1caede26fe2b4d57b208bd0174c386ad6ae01e54acd7bb422243c6055058f4231e250050134283a76de8eff

    Root/0(N)/2147483647(H)/1(N)/2147483646(H):
        selector: feffff7f
        xprv:     386014c6dfeb8dadf62f0e5acacfbf7965d5746c8b9011df155a31df7be0fb59986c923d979d89310acd82171dbaa7b73b20b2033ac6819d7f309212ff3fbabd
        xpub:     9f66aa8019427a825dd72a13ce982454d99f221c8d4874db59f52c2945cbcabd986c923d979d89310acd82171dbaa7b73b20b2033ac6819d7f309212ff3fbabd

    Root/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N):
        selector: 02
        xprv:     08c3772f5c0eee42f40d00f4faff9e4c84e5db3c4e7f28ecb446945a1de1fb59ef9d0a352f3252ea673e8b6bd31ac97218e019e845bdc545c268cd52f7af3f5d
        xpub:     67388f59a7b62644c3c6148575770e56969d77244530263bc9659b8563d7ff81ef9d0a352f3252ea673e8b6bd31ac97218e019e845bdc545c268cd52f7af3f5d


## Acknowledgements

We thank Dmitry Khovratovich and Jason Law for thorough analysis of the previous version of this scheme and their proposal [BIP32-Ed25519](https://drive.google.com/open?id=0ByMtMw2hul0EMFJuNnZORDR2NDA) where derived keys are also safe to use in ECDH implementations using Montgomery Ladder. We improve on their proposal further by slighly reducing the collision probability of child keys, reducing the size of xprv from 96 to 64 bytes, and using the extensible output hash function SHAKE128 instead of HMAC-SHA512.

We also thank Gregory Maxwell and Pieter Wuille for clarifying the design decisions behind [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki) and the capability of selectively proving linkage between arbitrary child keys.

Finally, we thank all participants on the [Curves](https://moderncrypto.org/mail-archive/curves/2017/000858.html) and [CFRG](https://www.ietf.org/mail-archive/web/cfrg/current/msg09077.html) mailing lists: Henry de Valence, Mike Hamburg, Trevor Perrin, Taylor R. Campbell and others.


## References

1. Hierarchical Deterministic Wallets, [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki)
2. EdDSA, [RFC 8032](https://tools.ietf.org/html/rfc8032)
3. HMAC-SHA512, [RFC 4231](http://tools.ietf.org/html/rfc4231)
4. [BIP32-Ed25519](https://drive.google.com/open?id=0ByMtMw2hul0EMFJuNnZORDR2NDA)


