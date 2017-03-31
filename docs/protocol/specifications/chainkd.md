# Chain Key Derivation

* [Introduction](#introduction)
* [Definitions](#definitions)
* [Algorithms](#algorithms)
  * [Generate root key](#generate-root-key)
  * [Generate extended public key](#generate-extended-public-key)
  * [Derive hardened extended private key](#derive-hardened-extended-private-key)
  * [Derive non-hardened extended private key](#derive-non-hardened-extended-private-key)
  * [Derive non-hardened extended public key](#derive-non-hardened-extended-public-key)
  * [Derive EdDSA public key](#derive-eddsa-public-key)
  * [Derive EdDSA secret key](#derive-eddsa-secret-key)
  * [Generate scalar](#generate-scalar)
  * [Encode public key](#encode-public-key)
* [FAQ](#faq)
* [Security](#security)
* [Test vectors](#test-vectors)
* [References](#references)

## Introduction

ChainKD is a deterministic key derivation scheme useful for digital signatures and Diffie-Hellman key exchange.

Features:

1. Scheme is fully deterministic and allows producing complex hierarchies of keys from a single high-entropy seed.
2. Derive private keys from extended private keys using “hardened derivation”.
3. Derive public keys independently from private keys using “non-hardened derivation”.
4. Hardened and non-hardened public keys and signatures are compatible with [EdDSA](https://tools.ietf.org/html/rfc8032) specification. Specifically, private keys are 64-byte raw secret keys used in [NaCl](https://nacl.cr.yp.to/sign.html) function `crypto_sign`.
5. Variable-length string selectors instead of fixed-length integer indexes.
6. Short 64-byte extended public and private keys without special encoding.
7. No metadata: an extended key carries only an additional 32-byte salt to avoid having the derivation function depend only on the key itself.
8. Privacy: extended key does not reveal neither its place in the hierarchy, nor the manner of derivation (hardened or non-hardened).

Limitations:

1. Depth of non-hardened derivation is limited to 2<sup>20</sup>.
2. Number of distinct root keys or hardened public keys is 2<sup>250</sup>, half of the keyspace allowed in EdDSA.
3. Number of distinct non-hardened public keys is 2<sup>230</sup> (reduced to keep compatibility with ECDH use and allow a comfortably large number of derivation levels).
4. Entropy of the nonce for any derived key is at least 2<sup>250</sup> which is 4 times lower than nonce in EdDSA.


## Definitions

**Hash(X,N)** is SHAKE-128 as specified in [FIPS 202](http://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.202.pdf) with input string `X` and output length `N` in bytes.

**Selector** is a variable-length byte string that can be used as a derivation index.

**Secret scalar** is 32-byte string representing a 256-bit integer using little-endian convention.

**Extended private key** (aka “xprv”) is a 64-byte string representing a key that can be used for deriving *child extended private and public keys*.

**Extended public key** (aka “xpub”) is a 64-byte string representing a key that can be used for deriving *child extended public keys*.

**EdDSA secret key** (aka “sk”) is a 64-byte string representing a raw secret key used for creating EdDSA signatures (consists of 32-byte scalar and 32-byte “prefix”).

**EdDSA public key** (aka “pk”) is a 32-byte string representing encoding of an elliptic curve point on Ed25519 as defined in EdDSA ([RFC 8032](https://tools.ietf.org/html/rfc8032)).


## Algorithms

### Generate root key

**Input:** `seed`, a seed byte sequence (variable-length, should contain at least 256 bits of entropy).

**Output:** `xprv`, a root extended private key.

1. Compute `K = Hash("ChainKD seed" || seed, 64)`.
2. Split `K` in two parts: 32-byte `buf` and 32-byte `spice`.
3. Clear the third highest bit of the last byte of `buf`.
4. [Generate scalar](#generate-scalar) `s` from buffer `buf`.
5. Let `privkey` be a 32-byte string encoding scalar `s` using little-endian convention.
6. Return `xprv = privkey || spice` (64 bytes).



### Generate extended public key

**Input:** `xprv`, an extended private key.

**Output:** `xpub`, an extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `spice`.
2. Interpret `privkey` as a little-endian 256-bit integer `s`.
3. Perform a fixed-base scalar multiplication `P = s·B` where `B` is a base point of Ed25519.
4. [Encode](#encode-public-key) point `P` as `pubkey`.
5. Compute `salt` as `spice` with the first byte set to zero.
6. Return extended public key `xpub = pubkey || salt` (64 bytes).


### Derive hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `spice`.
2. Compute `K = Hash(0x00 || privkey || spice || selector, 64)`.
3. Split `K` in two parts: 32-byte `buf` and 32-byte `spice’`.
4. Clear the third highest bit of the last byte of `buf`.
5. [Generate scalar](#generate-scalar) `s’` from buffer `buf`.
6. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
7. Return `xprv’ = privkey’ || spice’`.


### Derive non-hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `spice`.
2. Let `s` be the scalar decoded from `privkey` using little-endian notation.
3. Perform a fixed-base scalar multiplication `P = s·B` where `B` is a base point of Ed25519.
4. [Encode](#encode-public-key) point `P` as `pubkey`.
5. Compute `salt` as `spice` with the first byte set to zero.
6. Compute `F = Hash(0x01 || pubkey || salt || selector, 61)`.
7. Split `F` in two parts: 29-byte `fbuffer` and 32-byte `salt’`.
8. Clear top 2 bits of `fbuffer` and interpret it as a scalar `f` using little-endian notation.
9. Compute derived secret scalar `s’ = (s + 8·f) mod L` (where `L` is the group order of base point `B`).
10. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
11. Compute `pepper’ = Hash(0x02 || xprv || selector, 1)`.
12. Compute `spice’` as `salt’` with the first byte set to `pepper’`.
13. Return `xprv’ = privkey’ || spice’`.


### Derive non-hardened extended public key

**Inputs:**

1. `xpub`, an extended public key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xpub’`, the derived extended public key.

1. Split `xpub` in two parts: 32-byte `pubkey` and 32-byte `salt`.
2. Compute `F = Hash(0x01 || pubkey || salt || selector, 61)`.
3. Split `F` in two parts: 29-byte `fbuffer` and 32-byte `salt’`.
4. Clear top 2 bits of `fbuffer` and interpret it as a scalar `f` using little-endian notation.
5. Perform a fixed-base scalar multiplication `F = f·B` where `B` is a base point of Ed25519.
6. Decode point `P` from `pubkey` according to EdDSA.
7. Perform point addition `P’ = P + 8·F`.
8. [Encode](#encode-public-key) point `P’` as `pubkey’`.
9. Return `xpub’ = pubkey’ || salt’`.


### Derive EdDSA public key

**Input:** `xpub`, an extended public key.

**Output:** `pubkey`, a 32-byte [EdDSA](https://tools.ietf.org/html/rfc8032) public key.

1. Return first 32 bytes of `xpub` as encoded `pubkey` suitable for ECDH key exchange or EdDSA signatures.

Resulting 32-byte public key can be used to verify EdDSA signature created by a corresponding [EdDSA secret key](#derive-eddsa-secret-key).


### Derive EdDSA secret key

**Input:** `xprv`, an extended private key.

**Output:** `secretkey`, a 64-byte [EdDSA](https://tools.ietf.org/html/rfc8032) secret key.

1. Compute 32-byte hash `ext = Hash(0x03 || xprv, 32)`.
2. Extract `privkey` as first 32 bytes of `xprv`.
3. Return `secretkey = privkey || ext`.

Resulting 64-byte secret key can be used to create EdDSA signature verifiable by a corresponding [EdDSA public key](#derive-eddsa-public-key).


### Generate scalar

**Input:** `buffer`, a 32-byte string.

**Output:** `s`, a 256-bit integer.

1. Prune the buffer `buffer`: 
    1. the lowest 3 bits of the first byte are cleared, 
    2. the highest bit of the last byte is cleared, 
    3. the second highest bit of the last byte is set.
2. Interpret the buffer as the little-endian integer, forming a secret scalar `s`.
3. Return `s`.


### Encode public key

**Input:** `P`, a Ed25519 curve point.

**Output:** `pubkey`, a 32-byte string representing a point on Ed25519 curve.

1. First encode the y coordinate (in the range 0 <= y < p) as a little-endian string of 32 bytes. The most significant bit of the final byte is always zero.
2. To form the encoding of the point `P`, copy the least significant bit of the x coordinate to the most significant bit of the final byte. 
3. Return the resulting 32-byte string as `pubkey`.


## FAQ

**Can I derive hardened keys from non-hardened ones?**

Yes. The derivation method only affects relationship between the key and its parent, but does not affect how other keys are derived from that key.
Note that secrecy of all derived private keys (both hardened and non-hardened, at all levels) from a non-hardened key depends on keeping either the parent extended public key secret, or all non-hardened sibling keys secret.

**Is it safe to derive secret key from the scalar instead of raw 256 bits of entropy?**

EdDSA defines private key as raw 256 bits of entropy that are expanded using a hash function to a 512 bits: half of them are pruned to form a secret scalar that defines public key, another half is used to generate the nonce for the Schnorr signature. Unfortunately, 

We believe the scheme is equivalent to RFC6979 that derives the nonce by hashing the secret scalar. As an extra safety measure, the secret scalar is concatenated with the `salt` (which is not considered secret in this scheme) in order to make derivation function not dependent solely on the key.

**Why this scheme uses variable-length selectors instead of 31-bit indices as in BIP32?**

In our experience index-based derivation is not always convenient and can be extended to longer selectors only through additional derivation levels which is less efficient (e.g. 128-bit selectors would require 5 scalar multiplications in BIP32). However, users are free to use integer selectors by simply encoding them as 32-bit or 64-bit integers and passing to ChainKD. If you need to mix integer- and string-based indexing, you could prepend a type byte or use a standard encoding such as [Protocol Buffers](https://developers.google.com/protocol-buffers/) or [JSON](http://www.json.org).

**What is spice?**

**Spice** is a combination of **salt** and **pepper**: right 32 bytes of the extended private key consist of the 1 byte of pepper and 31 bytes of salt.

**What is salt?**

**Salt** is non-secret additional entropy stored in extended private and public keys that ensures that a derived key does not depend solely on the parent key.

**What is pepper?**

In ChainKD **pepper** is additional secret 8 bits of entropy (as opposed to non-secret **salt** which is part of the extended public key). Pepper is used to improve entropy of the nonce derived from the private key to match security guarantee of EdDSA.



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

Non-hardened derivation consist of adding scalars less that 2<sup>230</sup> (multiplied by 8) to a root private key. The resulting keys have the entropy of the root key (250 bits), but the number of possible public keys is reduced to 2<sup>230</sup> to allow large number of derivation levels. This means collisions of public keys are expected after deriving 2<sup>115</sup> keys.

### Secret scalar compatibility

EdDSA requires specific values for 5 bits of the secret scalar: lower 3 bits must be zero, higher 2 bits must be 1 and 0.

By setting high three bits of a root key (or hardened private key) to `010` and low three bits to `000`, that key has form `r = 2^254 + 8·k`, where `k < 2^250`. Each derived key `f` is generated from 230 bits and therefore less than `2^230`. Non-hardened scalar at level `i` is less than `2^254 + 2^253 + i·8·2^230`. Since the maximum depth is 2<sup>20</sup>, `i ≤ 2^20`, secret scalars at all levels are less than `2^254 + 2^253 + 2^252` and at the same divisible by 8. 

TBD: i think we need to shave off 1 more bit from 29-byte fbuffer to avoid overflow at level 2^20.

We will note that the depth limit is effectively reset at each level where hardened derivation is used.

### Nonce entropy

EdDSA derives a 64-byte private key (consisting of a secret scalar and a prefix used to generate a nonce) from 256 bits of entropy. In ChainKD extended private key carries the secret scalar as-is, together with 1 byte of _pepper_ (secret entropy) and 31 bytes of _salt_ (non-secret entropy). The 64-byte private key as required by EdDSA consists of a secret scalar (unmodified) and additional 32 bytes of _prefix_ used to generate nonce for the signature. 

In ChainKD that prefix is derived non-linearly from the extended private key, having combined entropy of both the secret scalar (250 bits) and pepper (8 bits). Additional bits of pepper therefore ensure that the nonce has at least 256 bits of randomness. While the prefix is not derived in parallel to secret scalar, the construction is similar to the one in [RFC6979](https://tools.ietf.org/html/rfc6979) where nonce is also computed from a secret scalar and a message.



## Test vectors

All values use hexadecimal encoding.

### ChainKD2 test vector 1

    Master:
        seed:     010203
        xprv:     TBD
        xpub:     TBD
            
    Master/010203(H):
        selector: 010203
        xprv:     TBD
        xpub:     TBD
        
    Master/010203(N):
        selector: 010203
        xprv:     TBD
        xpub:     TBD
    
    Master/010203(H)/""(N):
        selector: (empty string)
        xprv:     TBD
        xpub:     TBD
        
    Master/010203(N)/""(H):
        selector: (empty string)
        xprv:     TBD
        xpub:     TBD
    
    Master/010203(N)/""(N):
        selector: (empty string)
        xprv:     TBD
        xpub:     TBD


### ChainKD2 test vector 2

    Master:
        seed:     fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542
        xprv:     TBD
        xpub:     TBD
        
    Master/0(N):
        selector: 00
        xprv:     TBD
        xpub:     TBD
    
    Master/0(N)/2147483647(H):
        selector: ffffff7f
        xprv:     TBD
        xpub:     TBD
        
    Master/0(N)/2147483647(H)/1(N):
        selector: 01
        xprv:     TBD
        xpub:     TBD
    
    Master/0(N)/2147483647(H)/1(N)/2147483646(H):
        selector: feffff7f
        xprv:     TBD
        xpub:     TBD
        
    Master/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N):
        selector: 02
        xprv:     TBD
        xpub:     TBD


## Acknowledgements

We thank Dmitry Khovratovich and Jason Law for thorough analysis of the previous version of this scheme and their proposal [BIP32-Ed25519](https://drive.google.com/open?id=0ByMtMw2hul0EMFJuNnZORDR2NDA) where derived keys are also safe to use in ECDH implementations using Montgomery Ladder.


## References

1. Hierarchical Deterministic Wallets, [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki)
2. EdDSA, [RFC 8032](https://tools.ietf.org/html/rfc8032)
3. HMAC-SHA512, [RFC 4231](http://tools.ietf.org/html/rfc4231)


