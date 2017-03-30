# Chain Key Derivation

* [Introduction](#introduction)
* [Definitions](#definitions)
* [Security](#security)
* [Algorithms](#algorithms)
  * [Generate root key](#generate-root-key)
  * [Generate extended public key](#generate-extended-public-key)
  * [Derive hardened extended private key](#derive-hardened-extended-private-key)
  * [Derive non-hardened extended private key](#derive-non-hardened-extended-private-key)
  * [Derive non-hardened extended public key](#derive-non-hardened-extended-public-key)
  * [Sign](#sign)
  * [Verify signature](#verify-signature)
  * [Generate scalar](#generate-scalar)
  * [Encode public key](#encode-public-key)
* [FAQ](#faq)
* [Test vectors](#test-vectors)
* [References](#references)

## Introduction

This is a simple deterministic key derivation scheme useful for digital signatures (such as EdDSA) and Diffie-Hellman key exchange via Montogmery Ladder.

Features:

1. Scheme is fully deterministic and allows producing complex hierarchies of keys from a single high-entropy seed.
2. Derive private keys from extended private keys using “hardened derivation”.
3. Derive public keys independently from private keys using “non-hardened derivation”.
4. Hardened and non-hardened public keys and signatures are compatible with [EdDSA][RFC 8032] specification.
5. Variable-length string selectors instead of fixed-length integer indexes.
6. Short 64-byte extended public and private keys without special encoding.
7. No metadata: an extended key carries only an additional 32-byte salt to avoid having the derivation function depend only on the key itself.
8. Privacy: extended key does not reveal neither its place in the hierarchy, nor the manner of derivation (hardened or non-hardened).

Limitations:

1. Depth of non-hardened derivation is limited to 2<sup>20</sup>.
2. Number of distinct non-hardened public keys is 2<sup>230</sup>, while the number of distinct hardened public keys is 2<sup>251</sup> as in EdDSA.

## Definitions

**Hash(X,N)** is SHAKE-128 as specified in [FIPS 202](http://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.202.pdf) with input string `X` and output length `N` in bytes.

**Selector** is a variable-length byte string that can be used as a derivation index.

**Secret scalar** is 32-byte string representing a 256-bit integer using little-endian convention.

**Public key** is a 32-byte string representing a point on elliptic curve Ed25519 [RFC 8032].

**Extended private key** (aka “xprv”) is a 64-byte string representing a key that can be used for deriving *child extended private and public keys*.

**Extended public key** (aka “xpub”) is a 64-byte string representing a key that can be used for deriving *child extended public keys*.


## Security

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



## Algorithms

### Generate root key

**Input:** `seed`, a seed byte sequence (variable-length, should contain at least 256 bits of entropy).

**Output:** `xprv`, a root extended private key.

1. Calculate `I = Hash("ChainKD seed" || seed, 64)`.
2. Split `I` in two parts: 32-byte `buf` and 32-byte `salt`.
3. Clear the third highest bit of the last byte of `buf`.
4. [Generate scalar](#generate-scalar) `s` from buffer `buf`.
5. Let `privkey` be a 32-byte string encoding scalar `s` using little-endian convention.
6. Return `xprv = privkey || salt` (64 bytes).



### Generate extended public key

**Input:** `xprv`, an extended private key.

**Output:** `xpub`, an extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Interpret `privkey` as a little-endian 256-bit integer `s`.
3. Perform a fixed-base scalar multiplication `P = s·B` where `B` is a base point of Ed25519.
4. [Encode](#encode-public-key) point `P` as `pubkey`.
5. Return extended public key `xpub = pubkey || salt` (64 bytes).


### Derive hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Let `I = Hash(0x00 || privkey || salt || selector, 64)`.
3. Split `I` in two parts: 32-byte `buf` and 32-byte `salt’`.
4. Clear the third highest bit of the last byte of `buf`.
5. [Generate scalar](#generate-scalar) `s’` from buffer `buf`.
6. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
7. Return `xprv’ = privkey’ || salt’`.


### Derive non-hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Let `s` be the scalar decoded from `privkey` using little-endian notation.
3. Perform a fixed-base scalar multiplication `P = s·B` where `B` is a base point of Ed25519.
4. [Encode](#encode-public-key) point `P` as `pubkey`.
5. Let `I = Hash(0x01 || pubkey || salt || selector, 61)`.
6. Split `I` in two parts: 29-byte `fbuffer` and 32-byte `salt’`.
7. Clear top 2 bits of `fbuffer` and interpret it as a scalar `f` using little-endian notation.
8. Compute derived secret scalar `s’ = (s + 8·f) mod L` (where `L` is the group order of base point `B`).
9. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
10. Return `xprv’ = privkey’ || salt’`.


### Derive non-hardened extended public key

**Inputs:**

1. `xpub`, an extended public key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xpub’`, the derived extended public key.

1. Split `xpub` in two parts: 32-byte `pubkey` and 32-byte `salt`.
2. Let `len` be the length of `selector` in bytes.
3. Let `I = Hash(0x01 || pubkey || salt || selector, 61)`.
4. Split `I` in two parts: 29-byte `fbuffer` and 32-byte `salt’`.
5. Clear top 2 bits of `fbuffer` and interpret it as a scalar `f` using little-endian notation.
6. Perform a fixed-base scalar multiplication `F = f·B` where `B` is a base point of Ed25519.
7. Decode point `P` from `pubkey` according to EdDSA.
8. Perform point addition `P’ = P + 8·F`.
9. [Encode](#encode-public-key) point `P’` as `pubkey’`.
10. Return `xpub’ = pubkey’ || salt’`.


### Expand signing key

**Input:** `xprv`, an extended private key.

**Output:** `privkey`, an 64-byte key as used by EdDSA.

TBD.


### Sign

**Inputs:**

1. `xprv`, an extended private key.
2. `message`, a variable-length byte sequence representing a message to be signed.

**Output:** `(R,S)`, 64-byte string representing an EdDSA signature.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Let `s` be the scalar decoded from `privkey` using little-endian notation.
3. Let `h = Hash512(0x02 || privkey || salt)`.
4. Let `prefix` be the first half of `h`: `prefix = h[0:32]`.
5. Perform a fixed-base scalar multiplication `P = s·B` where `B` is a base point of Ed25519.
6. [Encode](#encode-public-key) point `P` as `pubkey`.
7. Compute `Hash512(prefix || message)`. Interpret the 64-byte digest as a little-endian integer `r`.
8. Compute the point `r·B`.  For efficiency, do this by first reducing `r` modulo `L`, the group order of `B`. 
9. Let the string `R` be the encoding of the point `r·B`.
10. Compute `Hash512(R || pubkey || message)`, and interpret the 64-byte digest as a little-endian integer `k`.
11. Compute `S = (r + k * s) mod L`. For efficiency, again reduce `k` modulo `L` first.
12. Concatenate `R` (32 bytes) and the little-endian encoding of `S` (32 bytes, three most significant bits of the final byte are always zero).
13. Return `(R,S)` (64 bytes).


### Verify signature

**Inputs:**

1. `xpub`, an extended public key.
2. `message`, a variable-length byte sequence representing a signed message.
3. `(R,S)`, a 64-byte signature.

**Output:** boolean value indicating if the signature is valid or not.

1. Extract public key `pubkey` as first 32 bytes of `xpub`.
2. Verify the EdDSA signature `(R,S)` over `message` using `pubkey` per [RFC 8032] substituting SHA512 hash function with Hash512 (which equals SHA512 in ChainKD-SHA2 instance thus retaining full compatibility with EdDSA verification procedure).


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

**BIP32 is fully compatible with ECDSA. Why this scheme does not follow standard EdDSA?**

EdDSA treats private key not as a raw scalar (which is what ECDSA does), but as a buffer being hashed and then split into a scalar and a `prefix` material for the nonce. This hashing creates a non-linear relationship that is impossible to map to curve points that only support linear operations for non-hardened derivation. This scheme therefore deviates from EdDSA and encodes a non-hardened private key as a scalar directly, without its hash preimage. For consistency, the hardened key also stores only the scalar, not its preimage. At the same time, signature verification is fully compatible with EdDSA for both hardened and non-hardened public keys.

**Is it safe to derive signature nonce directly from the secret scalar?**

We believe the scheme is equivalent to RFC6979 that derives the nonce by hashing the secret scalar. As an extra safety measure, the secret scalar is concatenated with the `salt` (which is not considered secret in this scheme) in order to make derivation function not dependent solely on the key.



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

We thank Dmitry Khovratovich and Jason Law for thorough analysis of the previous version of this scheme and their proposal [BIP32-Ed25519](https://drive.google.com/open?id=0ByMtMw2hul0EMFJuNnZORDR2NDA) which also takes into account use in ECDH.


## References

1. Hierarchical Deterministic Wallets, [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki)
2. EdDSA, [RFC 8032](https://tools.ietf.org/html/rfc8032)
3. HMAC-SHA512, [RFC 4231](http://tools.ietf.org/html/rfc4231)


