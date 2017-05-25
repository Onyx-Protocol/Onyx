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
  * [Generate secret scalar](#generate-secret-scalar)
  * [Encode public key](#encode-public-key)
* [FAQ](#faq)
* [Test vectors](#test-vectors)
* [References](#references)

## Introduction

This is a simple deterministic key derivation scheme consisting of two instances using different hash functions:

**ChainKD2:** the hash function is SHA2-512 (as in EdDSA specification).

**ChainKD3:** the hash function is SHA3-512.

Features:

1. Scheme is fully deterministic and allows producing complex hierarchies of keys from a single high-entropy seed.
2. Derive private keys from extended private keys using “hardened derivation”.
3. Derive public keys independently from private keys using “non-hardened derivation”.
4. Hardened and non-hardened public keys and signatures are compatible with [EdDSA][RFC 8032] specification.
5. Variable-length string selectors instead of fixed-length integer indexes.
6. Short 64-byte extended public and private keys without special encoding.
7. No metadata: an extended key carries only an additional 32-byte salt to avoid having the derivation function depend only on the key itself.
8. Privacy: extended key does not reveal neither its place in the hierarchy, nor the manner of derivation (hardened or non-hardened).


## Definitions

**Hash512** is a cryptographic hash function with 512-bit output (either SHA2-512 or SHA3-512).

**Selector** is a variable-length byte string that can be used as a derivation index.

**Secret scalar** is 32-byte string representing a 256-bit integer using little-endian convention.

**Public key** is a 32-byte string representing a point on elliptic curve Ed25519 [RFC 8032].

**Extended private key** (aka “xprv”) is a 64-byte string representing a key that can be used for deriving *child extended private and public keys*.

**Extended public key** (aka “xpub”) is a 64-byte string representing a key that can be used for deriving *child extended public keys*.

**LEB128** is a [Little Endian Base-128](https://developers.google.com/protocol-buffers/docs/encoding#varints) encoding for unsigned integers used for length-prefixing of a variable-length *selector* string.


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

1. Calculate `I = Hash512("Chain seed" || seed)`.
2. Split `I` in two parts: 32-byte `buf` and 32-byte `salt`.
3. [Generate secret scalar](#generate-secret-scalar) `s` from buffer `buf`.
4. Let `privkey` be a 32-byte string encoding scalar `s` using little-endian convention.
5. Return `xprv = privkey || salt` (64 bytes).



### Generate extended public key

**Input:** `xprv`, an extended private key.

**Output:** `xpub`, an extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Interpret `privkey` as a little-endian 256-bit integer `s`.
3. Perform a fixed-base scalar multiplication `P = s*B` where `B` is a base point of Ed25519.
4. [Encode](#encode-public-key) point `P` as `pubkey`.
5. Return extended public key `xpub = pubkey || salt` (64 bytes).


### Derive hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Let `len` be the length of `selector` in bytes.
3. Let `I = Hash512(0x00 || privkey || salt || LEB128(len) || selector)`.
4. Split `I` in two parts: 32-byte `buf` and 32-byte `salt’`.
5. [Generate secret scalar](#generate-secret-scalar) `s’` from buffer `buf`.
6. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
7. Return `xprv’ = privkey’ || salt’`.


### Derive non-hardened extended private key

**Inputs:**

1. `xprv`, an extended private key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xprv’`, the derived extended public key.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Let `s` be the scalar decoded from `privkey` using little-endian notation.
3. Perform a fixed-base scalar multiplication `P = s*B` where `B` is a base point of Ed25519.
4. [Encode](#encode-public-key) point `P` as `pubkey`.
5. Let `len` be the length of `selector` in bytes.
6. Let `I = Hash512(0x01 || pubkey || salt || LEB128(len) || selector)`.
7. Split `I` in two parts: 32-byte `fbuffer` and 32-byte `salt’`.
8. [Generate secret scalar](#generate-secret-scalar) `f` from buffer `fbuffer`.
9. Compute derived secret scalar `s’ = (f + s) mod L` (where `L` is the group order of `B`).
10. Let `privkey’` be a 32-byte string encoding scalar `s’` using little-endian convention.
11. Return `xprv’ = privkey’ || salt’`.


### Derive non-hardened extended public key

**Inputs:**

1. `xpub`, an extended public key.
2. `selector`, a variable-length byte sequence used as a derivation key.

**Output:** `xpub’`, the derived extended public key.

1. Split `xpub` in two parts: 32-byte `pubkey` and 32-byte `salt`.
2. Let `len` be the length of `selector` in bytes.
3. Let `I = Hash512(0x01 || pubkey || salt || LEB128(len) || selector)`.
4. Split `I` in two parts: 32-byte `fbuffer` and 32-byte `salt’`.
5. [Generate secret scalar](#generate-secret-scalar) `f` from buffer `fbuffer`.
6. Perform a fixed-base scalar multiplication `F = f*B` where `B` is a base point of Ed25519.
7. Decode point `P` from `pubkey` according to EdDSA.
8. Perform point addition `P’ = P + F`.
9. [Encode](#encode-public-key) point `P’` as `pubkey’`.
10. Return `xpub’ = pubkey’ || salt’`.


### Sign

**Inputs:**

1. `xprv`, an extended private key.
2. `message`, a variable-length byte sequence representing a message to be signed.

**Output:** `(R,S)`, 64-byte string representing an EdDSA signature.

1. Split `xprv` in two parts: 32-byte `privkey` and 32-byte `salt`.
2. Let `s` be the scalar decoded from `privkey` using little-endian notation.
3. Let `h = Hash512(0x02 || privkey || salt)`.
4. Let `prefix` be the first half of `h`: `prefix = h[0:32]`.
5. Perform a fixed-base scalar multiplication `P = s*B` where `B` is a base point of Ed25519.
6. [Encode](#encode-public-key) point `P` as `pubkey`.
7. Compute `Hash512(prefix || message)`. Interpret the 64-byte digest as a little-endian integer `r`.
8. Compute the point `r*B`.  For efficiency, do this by first reducing `r` modulo `L`, the group order of `B`. 
9. Let the string `R` be the encoding of the point `r*B`.
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


### Generate secret scalar

**Input:** `buffer`, a 32-byte string.

**Output:** `s`, a 256-bit integer.

1. Prune the buffer `buffer`: the lowest 3 bits of the first byte are cleared, the highest bit of the last byte is cleared, and the second highest bit of the last byte is set.
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
        xprv:     e892d064d9658a3405e97f5dfaefab9b3a08a2341cdeb427ae7d6f2eb96b3952967a0ec62a845bccb318935c012f6900b330d2831f6407eb0dd7df1082c2e22b
        xpub:     254a6f2c96f84aabaef5f2922026360c03d29ce3eb3de739c8c243053e1a3cbe967a0ec62a845bccb318935c012f6900b330d2831f6407eb0dd7df1082c2e22b
            
    Master/010203(H):
        selector: 010203
        xprv:     209f3ae66a0ef7bef75497fd214b821133d44ff2f8eb80b50b738b3e9ec67f5f2b037c3ec24d503128664eb2e773c0c96b6e102faf898568177491188180bd4f
        xpub:     e844c655dfced878e489d42c3ea26b9877e1c7f8c2dbad679525f8056fa5cfba2b037c3ec24d503128664eb2e773c0c96b6e102faf898568177491188180bd4f
        
    Master/010203(N):
        selector: 010203
        xprv:     3e42fb09bd0b6360e51c9b7ab70d1010e53eca59be378764535b0143b3a0ca0e4ee9f0b88260285f0b93b6b115e8e978351e4f1491d622821d78cde389c44e28
        xpub:     061155751a79a3d7dda52a7ea9980bdb1d06bf793be6b78cc8f5724541d5b1c64ee9f0b88260285f0b93b6b115e8e978351e4f1491d622821d78cde389c44e28
    
    Master/010203(H)/""(N):
        selector: (empty string)
        xprv:     97ae121e2d8b7ca893406edd6d170f260c1d8282eceee975eeb506af2dfbc808dd979ffd561bd9e60cced900e878de425868e0c70b944f7421816fafb6e3b224
        xpub:     3eca1608be5fa17867bddccd2b99eef344097c6ba17f19b9f54604c77f196813dd979ffd561bd9e60cced900e878de425868e0c70b944f7421816fafb6e3b224
        
    Master/010203(N)/""(H):
        selector: (empty string)
        xprv:     981da97280c994c3c0f5fe1990a263bbaf5493576c98102e9a1dd635e728c65eff84c4ba93c29e42cc6f89981b6bd903c3b78f03fa6e9d694a123abcfe024357
        xpub:     bc6a0009d5249872e94e1058a95f226560ab9c218665e18f34b168dd45b70b41ff84c4ba93c29e42cc6f89981b6bd903c3b78f03fa6e9d694a123abcfe024357
    
    Master/010203(N)/""(N):
        selector: (empty string)
        xprv:     604e33854c66f785e05d36d774b0b3dbe1286526ab8ded41f0cbfe5dfbf68a0a6bd8b033689d38055b58baff8eccceb623871e9c23be82606e903f2d71304208
        xpub:     3f61a6f6e543ffaebf68c9a0c0d64498e03d048d658f8f06bf9a9b6b3ddcb16a6bd8b033689d38055b58baff8eccceb623871e9c23be82606e903f2d71304208


### ChainKD2 test vector 2

    Master:
        seed:     fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542
        xprv:     f06907ad9298c685a4fd250538605bea7fa387388954e15a90b337c4ac889e467730a16f62d5159c3a0d390a0e4639be86c766ad779c810458adb532164a9211
        xpub:     55b33d123033131c8642ef736b4b1bf9430f52dbcb3b7d6bbf721040cf504bd57730a16f62d5159c3a0d390a0e4639be86c766ad779c810458adb532164a9211
        
    Master/0(N):
        selector: 00
        xprv:     2cb4d70521f62eeedb0e2d68a6843431800b9271c83a49a9ba598f85b2229e0446fb34a28f8cc239bfc700c9002aca2d5f2affff27955de947a1b4d3e232b229
        xpub:     06820e5ee702c54efea0aeea41f89dab5dd82d0797bb79689dee1ebc1ac00a1646fb34a28f8cc239bfc700c9002aca2d5f2affff27955de947a1b4d3e232b229
    
    Master/0(N)/2147483647(H):
        selector: ffffff7f
        xprv:     98c4c05731fed5f944345bdec859403d26cf8825f358740db2c107f720a8d2704f785675bea750ef52c78e56d973b4d0638ce5b3e76a8957c2d2c45dafb87c95
        xpub:     a30818e3b50163b0f346eba0dfef70e66041b7de97273c1b8cb0804d4645f1d44f785675bea750ef52c78e56d973b4d0638ce5b3e76a8957c2d2c45dafb87c95
        
    Master/0(N)/2147483647(H)/1(N):
        selector: 01
        xprv:     67f882c251a541d68460934283f78c38eb94b1d1b85ca64ebbf860bdd63ded0b811476e6e32936d8d6164d9f28ec7a3278b24758433ebe7d74e0db8a56930aaf
        xpub:     437835c60770e2890bf622df3ee66c07ba8628ed87591fbe0907607888435178811476e6e32936d8d6164d9f28ec7a3278b24758433ebe7d74e0db8a56930aaf
    
    Master/0(N)/2147483647(H)/1(N)/2147483646(H):
        selector: feffff7f
        xprv:     08cb5d261af0d47b4dadfe4b21b71decc844249892644a3f892d79eb38a3dc4db1dcbf10a891e1c3c1e49e6d6d5bda12049501ddb8121a52d7ed5c6658c71bc0
        xpub:     80923c7d5bbf37a269c862764b14a53b751a9cb786bce7c3d463d899806014fdb1dcbf10a891e1c3c1e49e6d6d5bda12049501ddb8121a52d7ed5c6658c71bc0
        
    Master/0(N)/2147483647(H)/1(N)/2147483646(H)/2(N):
        selector: 02
        xprv:     6e9f9333156b5bb074456fdf75a2acb3d67a0b1dce044cf00efd331087719807574d3c263a60a4e40425032a89dd36bbf02fb98ccb9495bceaea1d1ad3d91973
        xpub:     cd4c4b318b65e0e85b6f00a0ed0c4591c96c6d89d128b0cc90497d39150c2428574d3c263a60a4e40425032a89dd36bbf02fb98ccb9495bceaea1d1ad3d91973


## References

1. [RFC 8032]
2. [BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki)
3. [LEB-128](https://developers.google.com/protocol-buffers/docs/encoding#varints)

[RFC 8032]: https://tools.ietf.org/html/rfc8032
