# Data Types Specification

* [Introduction](#introduction)
* [Definitions](#definitions)
  * [LEB128](#leb128)
  * [Integer](#integer)
  * [String](#string)
  * [SHA3](#sha3)
  * [Hash](#hash)
  * [List](#list)
  * [Struct](#struct)
  * [Public Key](#public-key)
  * [Signature](#signature)

## Introduction

This document describes the blockchain data structures used in the Chain Protocol.

## Definitions

### LEB128

[Little Endian Base 128](https://developers.google.com/protocol-buffers/docs/encoding) encoding for unsigned integers typically used to specify length prefixes for arrays and strings. Values in range [0, 127] are encoded in one byte. Larger values use two or more bytes.

### Integer

A LEB128 integer with a maximum allowed value of 0x7fffffffffffffff (2<sup>63</sup> – 1) and a minimum of 0. A varint63 fits into a signed 64-bit integer.

### String

A binary string with a LEB128 prefix specifying its length in bytes.
The maximum allowed length of the underlying string is 0x7fffffff (2<sup>31</sup> – 1).

The empty string is encoded as a single byte 0x00, a one-byte string is encoded with two bytes 0x01 0xNN, a two-byte string is 0x02 0xNN 0xMM, etc. 

### SHA3

*SHA3* refers to the SHA3-256 function as defined in [FIPS202](https://dx.doi.org/10.6028/NIST.FIPS.202) with a fixed-length 32-byte output.

This hash function is used throughout all data structures and algorithms in this spec,
with the exception of SHA-512 (see [FIPS180](http://csrc.nist.gov/publications/fips/fips180-2/fips180-2withchangenotice.pdf)) used internally as function H inside Ed25519 (see [CFRG1](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05)).

### Hash

A fixed-length 32-byte string.

### List

A `List` is encoded as a [String](#string) containing the serialized items, one by one, as defined by the schema. 

Note: since the `List` is encoded as a variable-length string, its length prefix indicates not the number of _items_,
but the number of _bytes_ of all the items in their serialized form.

### Struct

A `Struct` is encoded as a concatenation of all its serialized fields.

### Public Key

In this document, a *public key* is the 32-byte binary encoding
of an Ed25519 (EdDSA) public key, as defined in [CFRG1](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05).

### Signature

In this document, a *signature* is the 64-byte binary encoding
of an Ed25519 (EdDSA) signature, as defined in [CFRG1](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05).


