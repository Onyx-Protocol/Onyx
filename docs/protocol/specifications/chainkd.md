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
  * [Encode public key](#encode-public-key)
* [FAQ](#faq)
* [Security](#security)
* [Test vectors](#test-vectors)
* [References](#references)

## Introduction

**ChainKD** is a deterministic key derivation scheme. Generated keys are compatible with EdDSA signature scheme and Diffie-Hellman key exchange.

Features:

1. Scheme is fully deterministic and allows producing complex hierarchies of keys from a single high-entropy seed.
2. Derive private keys from extended private keys using “hardened derivation”.
3. Derive public keys independently from private keys using “non-hardened derivation”.
4. Hardened and non-hardened public keys and signatures are compatible with [EdDSA](https://tools.ietf.org/html/rfc8032) specification. Signing keys are 64-byte strings, the format used in [NaCl](https://nacl.cr.yp.to/sign.html) function `crypto_sign`.
5. Variable-length string selectors instead of fixed-length 31-bit integer indices.
6. Short 64-byte extended public and private keys without special encoding.
7. No metadata: an extended key carries only an additional 32-byte salt that allows sharing public key without revealing child public keys.
8. Privacy: extended key does not reveal neither its place in the hierarchy, nor the manner of derivation (hardened or non-hardened).

Limitations:

1. Depth of non-hardened derivation is limited to 2<sup>20</sup> (more than million levels).
2. Number of distinct root keys or hardened public keys is 2<sup>250</sup>, half of the keyspace allowed in EdDSA.
3. Number of distinct non-hardened public keys is 2<sup>230</sup>.


## Definitions

**Selector** is a variable-length byte string indexing a child key during key derivation.

**Secret scalar** is 32-byte string representing a 256-bit integer using little-endian convention.

**Extended private key** (aka “xprv”) is a 64-byte string representing a key that can be used for deriving *child extended private and public keys*.

**Extended public key** (aka “xpub”) is a 64-byte string representing a key that can be used for deriving *child extended public keys*.

**Derivation key** (aka “dk”) is the second half (32-byte string) of the extended private or public key that enables derivation of the child keys or proving linkability of the keys.

**Signing key** (aka “sk”) is a 64-byte string representing a raw signing key used for creating EdDSA signatures (consists of 32-byte scalar and 32-byte “prefix”). This is the format used by `crypto_sign` function in [NaCl](https://nacl.cr.yp.to/sign.html) library.

**Private key** is a the first half of the _signing key_ representing a raw signing scalar usable for ECDH and generating a _public key_.

**Public key** (aka “pk”) is a 32-byte string representing encoding of an elliptic curve point on Ed25519 as defined in EdDSA ([RFC 8032](https://tools.ietf.org/html/rfc8032)).


## Algorithms

### Generate root key

**Input:** `seed`, a seed byte sequence (variable-length, should contain at least 256 bits of entropy).

**Output:** `xprv`, a root extended private key.

1. Compute 64-byte string `xprv = HMAC-SHA512(key: "Root", data: seed)`.
2. Prune the first 32 bytes of `xprv` to produce a valid scalar:
    1. the lowest 3 bits of the first byte are cleared,
    2. the highest bit of the last byte is cleared,
    3. the second highest bit of the last byte is set,
    4. the third highest bit of the last byte is cleared.
3. Return `xprv`.


### Generate extended public key

**Input:** `xprv`, an extended private key.

**Output:** `xpub`, an extended public key.

1. Split `xprv` in two halves: 32-byte scalar `s` and 32-byte derivation key `dk`.
2. Perform a fixed-base scalar multiplication `P = s·B` where `B` is a base point of Ed25519.
3. [Encode](#encode-public-key) point `P` as `pubkey`.
4. Return extended public key `xpub = pubkey || dk` (64 bytes).


### Derive hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two halves: 32-byte scalar `s` and 32-byte derivation key `dk`.
2. Compute `xprv’ = HMAC-SHA512(key: dk, data: "H" || s || selector)`.
3. Prune the first 32 bytes of `xprv’` to produce a valid scalar:
    1. the lowest 3 bits of the first byte are cleared,
    2. the highest bit of the last byte is cleared,
    3. the second highest bit of the last byte is set,
    4. the third highest bit of the last byte is cleared.
4. Return `xprv’`.


### Derive non-hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length derivation index.

**Output:** `xprv’`, the derived extended public key.

1. [Generate extended public key](#generate-extended-public-key) `xpub` for a given `xprv`.
2. Split `xpub` in two halves: 32-byte pubkey `P` and 32-byte derivation key `dk`.
3. Compute `F = HMAC-SHA512(key: dk, data: "N" || P || selector)`.
4. Split `F` in two halves: a 32-byte `fbuffer` and a 32-byte `dk’`.
5. Clear lowest 3 bits and highest 23 bits of `fbuffer` and interpret it as a scalar `f` using little-endian notation.
6. Compute derived secret scalar `s’ = s + f` (without reducing the result modulo the subgroup order).
7. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
8. Return `xprv’ = privkey’ || dk’`.


### Derive non-hardened extended public key

**Inputs:**

1. `xpub`, an extended public key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xpub’`, the derived extended public key.

1. Split `xpub` in two halves: 32-byte pubkey `P` and 32-byte derivation key `dk`.
2. Compute `F = HMAC-SHA512(key: dk, data: "N" || P || selector)`.
3. Split `F` in two halves: a 32-byte `fbuffer` and a 32-byte `dk’`.
4. Clear lowest 3 bits and highest 23 bits of `fbuffer` and interpret it as a scalar `f` using little-endian notation.
5. Perform a fixed-base scalar multiplication `F = f·B` where `B` is a base point of Ed25519.
6. Decode point `P` from `pubkey` according to EdDSA.
7. Perform point addition `P’ = P + F`.
8. [Encode](#encode-public-key) point `P’` as `pubkey’`.
9. Return `xpub’ = pubkey’ || dk’`.


### Extract public key

**Input:** `xpub`, an extended public key.

**Output:** `pubkey`, a 32-byte [EdDSA](https://tools.ietf.org/html/rfc8032) public key.

1. Return first 32 bytes of `xpub` as encoded `pubkey` suitable for ECDH key exchange or EdDSA signatures.

Resulting 32-byte public key can be used to verify EdDSA signature created by a corresponding [EdDSA signing key](#extract-signing-key).


### Extract signing key

**Input:** `xprv`, an extended private key.

**Output:** `sk`, a 64-byte [EdDSA](https://tools.ietf.org/html/rfc8032) signing key.

1. Compute hash `exthash = HMAC-SHA512(key: "Expand", data: xprv)`.
2. Extract `privkey` as first 32 bytes of `xprv`.
3. Extract `ext` as last 32 bytes of `exthash`.
4. Return 64-byte signing key `sk = privkey || ext`.

Resulting 64-byte signing key can be used to create EdDSA signature verifiable by a corresponding [EdDSA public key](#extract-public-key).


### Encode public key

**Input:** `P`, a Ed25519 curve point.

**Output:** `pubkey`, a 32-byte string representing a point on Ed25519 curve.

1. First encode the y coordinate (in the range 0 <= y < p) as a little-endian string of 32 bytes. The most significant bit of the final byte is always zero.
2. To form the encoding of the point `P`, copy the least significant bit of the x coordinate to the most significant bit of the final byte.
3. Return the resulting 32-byte string as `pubkey`.


## Design rationale

TBD:

* naming: xpub,xprv, dk vs chaincode, 
* torsion-safe representative by HdV et al is not used to keep full compatibility with existing codebases that might rely on the high bit set
* expanded privkey as in NaCl-2011 used for max compatibility with existing EdDSA codebases
* 2^20 depth chosen for comfortable max depth while keeping prob of collisions negligibly low.  (reduced to allow a comfortably large number of derivation levels while keeping strict compatibility with EdDSA and ECDH).


### Names

**ChainKD** stands for “”

### Deriving hardened keys from non-hardened ones

The derivation method only affects relationship between the key and its parent,
but does not affect how other keys are derived from that key.
Note that secrecy of all derived private keys (both hardened and non-hardened, 
at all levels) from a non-hardened key depends on keeping either the parent 
extended public key secret, or all non-hardened sibling keys secret.

### Variable-length selectors instead of integers

[BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki) uses
31-bit indices for derivation. ChainKD generalizes indices to arbitrary-length 
binary strings called “selectors”.

In our experience index-based derivation is not always convenient and can be 
extended to longer selectors only through additional derivation levels which 
is less efficient (e.g. 128-bit selectors would require 5 scalar multiplications 
in BIP32). However, users are free to use integer selectors by simply encoding 
them as 32-bit or 64-bit integers and passing to ChainKD. If you need to mix 
integer- and string-based indexing, you could prepend a type byte or use a standard 
encoding such as [Protocol Buffers](https://developers.google.com/protocol-buffers/) 
or [JSON](http://www.json.org).



## Security

### Capabilities

Knowledge of the seed or the root extended private key:

1. Allows deriving hardened extended private key.
2. Allows deriving non-hardened extended private key.
3. Allows signing messages with the root key.

Knowledge of an extended private key:

1. Allows deriving hardened extended private key.
2. Allows deriving non-hardened extended private key.
3. Allows signing messages with that key.

Knowledge of an extended public key:

1. Allows deriving non-hardened public keys.
2. Does not allow determining if it is derived in a hardened or non-hardened way.
3. Does not allow determining if another extended public key is a sibling of the key.
4. Does not allow signing
5. Does not allow deriving private keys.
6. Does not allow deriving hardened public keys.

Knowledge of a parent extended public key and one of non-hardened derived extended private keys:

1. Allows extracting parent private key: `s = (s’ - f) mod L` where `f` is derived from the parent `xpub` and `s’` is extracted from the child `xprv’`.

### Root key security

We set 6 bits in the secret 256 bits of a 512-bit root extended key. Therefore, the root key requires an order of 2<sup>250</sup> attempts by brute-force.

### Hardened derivation security

Private keys derived using hardened derivation have 6 bits set, just like the root key. Therefore, an extended private key requires an order of 2<sup>250</sup> attempts by brute-force.

### Non-hardened derivation security

Non-hardened derivation consist of adding scalars less that 2<sup>230</sup> (multiplied by 8) to a root private key. The resulting keys have the entropy of the root key (250 bits), but the number of possible public keys is reduced to 2<sup>230</sup> to allow large number of derivation levels. This means collisions of public keys are expected after deriving 2<sup>115</sup> keys. We note that increased probability of collisions does not reduce security of EdDSA signatures or ECDH key exchange; it only marginally reduces unlinkability safety in privacy schemes based on one-time keys.


### Secret scalar compatibility

EdDSA requires specific values for 5 bits of the secret scalar: lower 3 bits must be zero, higher 2 bits must be 1 and 0.

By setting high three bits of a root key (or hardened private key) to `010` and low three bits to `000`, that key has form `r = 2^254 + 8·k`, where maximum value  of `k` is `2^250 - 1`. Each non-hardened derived scalar `f` is generated from 230 bits and has maximum value `2^230 - 1`. Therefore a key at level `i` has maximum value `2^254 + 2^253 - 8 + i·8·(2^230 - 1)`. Since the maximum `i` equals `2^20`, maximum value of any key is `2^255 - 2^23 - 8`. As a result, all deriveable keys are less than `2^255`, larger than `2^254` and divisible by 8 as required by EdDSA.

The depth limit is reset at each level where hardened derivation is used.

### Nonce entropy

EdDSA derives a 64-byte signing key from 256 bits of entropy. In ChainKD the extended private key carries the secret scalar as-is and 32 bytes of _derivation key_. The 64-byte signing key as required by EdDSA consists of a secret scalar (unmodified) and additional 32 bytes of _prefix_ used to generate nonce for the signature.

In ChainKD that prefix is derived non-linearly from the extended private key, having the entropy of the secret scalar (250 bits). The prefix is not derived in parallel to secret scalar, but from it, making the construction similar to the one in [RFC6979](https://tools.ietf.org/html/rfc6979) where the nonce is also computed from a secret scalar and a message using an HKDF construction.



## Test vectors

All values use hexadecimal encoding.

### ChainKD test vector 1

    Root:
        seed:     010203
        xprv:     50f8c532ce6f088de65c2c1fbc27b491509373fab356eba300dfa7cc587b07483bc9e0d93228549c6888d3f68ad664b92c38f5ea8ca07181c1410949c02d3146
        xpub:     e11f321ffef364d01c2df2389e61091b15dab2e8eee87cb4c053fa65ed2812993bc9e0d93228549c6888d3f68ad664b92c38f5ea8ca07181c1410949c02d3146

    Root/010203(H):
        selector: 010203
        xprv:     98bd4126e9040d7dfcf6c4d1ceb674db0569e7f21266eebf3dc9a469bab1ab45200bd2d6a956e819c68134a40be13e2653ccdcbaab92f7fd492626886884f832
        xpub:     696809f6ac24c8b70dde8778a8a0db26f642388be12b6323f12a97fcc3cbccbb200bd2d6a956e819c68134a40be13e2653ccdcbaab92f7fd492626886884f832

    Root/010203(N):
        selector: 010203
        xprv:     30837f155673a659f5c659045b598b175ceea3724c07dc53910e8392df7b0748d40ba49ebee85271fd1d53a45bfbb228623e98c43227fd1484f17139736f2f39
        xpub:     2e457bd3bd135cbe5bd46821588ad82b74e8b9cb256e3a956d72322df61b51acd40ba49ebee85271fd1d53a45bfbb228623e98c43227fd1484f17139736f2f39

    Root/010203(H)/""(N):
        selector: (empty string)
        xprv:     0889925d37b9664af32c78cb8225022b5854586c08f8a9a7ed3a657279b2ab45ae8c6d29a2d80e7dc8a141058ff68c257e59c45daba3184b100456828ed9ade8
        xpub:     6b45415a0638feb47a5eab07961883fafe476b637de7004111317a2454465ae2ae8c6d29a2d80e7dc8a141058ff68c257e59c45daba3184b100456828ed9ade8

    Root/010203(N)/""(H):
        selector: (empty string)
        xprv:     b8b626e7ce7e86c7e673e5652de643b98631771bb1602136bdb154863e606e5c360b2aee72cb1b1d62eccba447c164629ea956758982ccbb0a1a26fc991b7fd2
        xpub:     174eba73de14f9af2693c63c16e3466577ffc4e780846c8ff81f69fd0346af83360b2aee72cb1b1d62eccba447c164629ea956758982ccbb0a1a26fc991b7fd2

    Root/010203(N)/""(N):
        selector: (empty string)
        xprv:     484148c20a28b663bc71d72e5f84df77e11ae9ac128d450b311635e6cd7c0748e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877
        xpub:     5786f826e7e09d17d6c1928952653275d81ad5fba728a16b5d0b04bd1ed9ee35e70c8fb4062f4e8b4829ab1788d4a2ca71e056044503d6adfa75b229fb03d877


### ChainKD test vector 2

    Root:
        seed:     fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542
        xprv:     0031615bdf7906a19360f08029354d12eaaedc9046806aefd672e3b93b024e495a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798
        xpub:     f153ef65bbfaec3c8fd4fceb0510529048094093cf7c14970013282973e117545a95ba63cf47903eb742cd1843a5252118f24c0c496e9213bd42de70f649a798

    Root/0(N):
        selector: 00
        xprv:     b0d0dcae67caf04caca70eae5da892f862ffda71a12eeef0c857250c7e024e49ccb779405d72758d7fd6a562a221bdd30c430424d0b6871bbb54dd070fbbe477
        xpub:     ef490a29370687a02e1915ddd583e13210b37882befb4381cc9dc14a488309acccb779405d72758d7fd6a562a221bdd30c430424d0b6871bbb54dd070fbbe477

    Root/0(N)/2147483647(H):
        selector: ffffff7f
        xprv:     787b2712f7dc8674040b212d0ef171dd96aa5c0c3df0104cf2bae8b224d442541c19a82372e94016d103187267ce952a988f80a371b061e4493619a52025ff01
        xpub:     d807d625e9b55c3c099a7d43853f80429146f6f02b53469c1182a6bd45836d021c19a82372e94016d103187267ce952a988f80a371b061e4493619a52025ff01

    Root/0(N)/2147483647(H)/1(N):
        selector: 01
        xprv:     a87166b647dceaac5723eebfdb3fde90007b38a60fd9de3d771ba9ef9ed442542f1ad730181f2a2c1160437dfcd71004c1a8cced671e7c9b772e35e1d47e0ae0
        xpub:     794473aa11f7148b634e94444332f3ccafbfcb43617bd2751fc113218b1999af2f1ad730181f2a2c1160437dfcd71004c1a8cced671e7c9b772e35e1d47e0ae0

    Root/0(N)/2147483647(H)/1(N)/2147483646(H):
        selector: feffff7f
        xprv:     d8abfb1acd80058f7c2590d92b4a4fac6d031fc73ec0229e326921df97987644d246e0ccb91215621d68b74b418ebdaece8b52b0d13fb1e47922910d8c30493f
        xpub:     11aa4807361528e2bca0f26914b570d84cee26f8603378aa4c36fd1b76ec78ead246e0ccb91215621d68b74b418ebdaece8b52b0d13fb1e47922910d8c30493f

    Root/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N):
        selector: 02
        xprv:     281e8c4aff2fdeb0018ef283b64755f99baf993e3acccee4d87484f21f99764443388d06e53716a83060d4df1cdccfe8364029cfdb9422d5bffc31732fdca243
        xpub:     7d152694a55b166f7038aa4aee0f5865c8e777caec8778c41b95d8997754f80343388d06e53716a83060d4df1cdccfe8364029cfdb9422d5bffc31732fdca243


## Acknowledgements

We thank Dmitry Khovratovich and Jason Law for thorough analysis of the previous version of this scheme and their proposal [BIP32-Ed25519](https://drive.google.com/open?id=0ByMtMw2hul0EMFJuNnZORDR2NDA) where derived keys are also safe to use in ECDH implementations using Montgomery Ladder. We improve on their proposal further by slighly reducing collision probability of child keys, reducing size of xprv from 96 to 64 bytes and using extensible output hash function SHAKE128 instead of HMAC-SHA512.

We also thank Gregory Maxwell and Pieter Wuille for clarifying design decisions behind [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki) and capability of selectively proving linkage between arbitrary child keys.

Finally, thanks to all participants on the [Curves](https://moderncrypto.org/mail-archive/curves/2017/000858.html) and [CFRG](https://www.ietf.org/mail-archive/web/cfrg/current/msg09077.html) mailing lists: Henry de Valence, Mike Hamburg, Trevor Perrin, Taylor R Campbell and others.


## References

1. Hierarchical Deterministic Wallets, [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki)
2. EdDSA, [RFC 8032](https://tools.ietf.org/html/rfc8032)
3. HMAC-SHA512, [RFC 4231](http://tools.ietf.org/html/rfc4231)


