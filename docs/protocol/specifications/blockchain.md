# Blockchain Specification

* [Introduction](#introduction)
  * [Block](#block)
  * [Transaction](#transaction)
* [Types](#types)
  * [LEB128](#leb128)
  * [Integer](#integer)
  * [String](#string)
  * [SHA3](#sha3)
  * [Hash](#hash)
  * [List](#list)
  * [Struct](#struct)
  * [Public Key](#public-key)
  * [Signature](#signature)
* [Auxiliary data structures](#auxiliary-data-structures)
  * [Extension Struct](#extension-struct)
  * [Pointer](#pointer)
  * [Program](#program)
  * [Asset Definition](#asset-definition)
  * [Asset ID](#asset-id)
  * [Asset Amount](#asset-amount-1)
  * [Value Source 1](#value-source-1)
  * [Value Destination 1](#value-destination-1)
  * [Merkle Root](#merkle-root)
  * [Merkle Binary Tree](#merkle-binary-tree)
  * [Merkle Patricia Tree](#merkle-patricia-tree)
  * [Transactions Merkle Root](#transactions-merkle-root)
  * [Assets Merkle Root](#assets-merkle-root)
* [Entries](#entries)
  * [Entry](#entry)
  * [Entry ID](#entry-id)
  * [Block Header](#block-header)
  * [Block ID](#block-id)
  * [Transaction Header](#transaction-header)
  * [Transaction ID](#transaction-id)
  * [Output 1](#output-1)
  * [Spend 1](#spend-1)
  * [Issuance 1](#issuance-1)
  * [Mux 1](#mux-1)
  * [Nonce](#nonce)
  * [Time Range](#time-range)


## Introduction

This is a specification of the semantic data structures used by blocks and transactions. These data structures and rules are used for validation and hashing. This format is independent from the format for transaction wire serialization.

### Block

A **block** is a [block header](#block-header) together with a list of [transactions](#transaction).

### Transaction

A **transaction** is composed of a set of [entries](#entries). Each transaction must include a [transaction header](#transaction-header), which references other entries in the transaction, which in turn can reference additional entries. 

Every entry is identified by its [Entry ID](#entry-id).


## Types

### LEB128

[Little Endian Base 128](https://developers.google.com/protocol-buffers/docs/encoding) encoding for unsigned integers typically used to specify length prefixes for arrays and strings. Values in range [0, 127] are encoded in one byte. Larger values use two or more bytes.

### Integer

A [LEB128](#leb128) integer with a maximum allowed value of 0x7fffffffffffffff (2<sup>63</sup> – 1) and a minimum of 0. A varint63 fits into a signed 64-bit integer.

### String

A binary string with a [LEB128](#leb128) prefix specifying its length in bytes.
The maximum allowed length of the underlying string is 0x7fffffff (2<sup>31</sup> – 1).

The empty string is encoded as a single byte 0x00, a one-byte string is encoded with two bytes 0x01 0xNN, a two-byte string is 0x02 0xNN 0xMM, etc. 

### SHA3

*SHA3* refers to the SHA3-256 function as defined in [FIPS202](https://dx.doi.org/10.6028/NIST.FIPS.202) with a fixed-length 32-byte output.

This hash function is used throughout all data structures and algorithms in this spec,
with the exception of SHA-512 (see [FIPS180](http://csrc.nist.gov/publications/fips/fips180-2/fips180-2withchangenotice.pdf)) used internally as function H inside Ed25519 (see [CFRG1](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05)).

### Hash

A fixed-length 32-byte string.

### List

A `List` is encoded as a [Varstring31](#string) containing the serialized items, one by one, as defined by the schema. 

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


## Auxiliary data structures

Auxiliary data structures are [Structs](#struct) that are not [entries](#entries) by themselves, but used as fields within the entries.

### Extension Struct

An `Extension Struct` is encoded as a single 32-byte hash. 
Future versions of the protocol may add additional fields as `Extension Structs` that will be compressed in a single hash for backwards compatibility.

### Pointer

A `Pointer` is encoded as a [Hash](#hash), and identifies another [entry](#entry) by its [ID](#entry-id). 

`Pointer` restricts the possible acceptable types: `Pointer<X>` must refer to an entry of type `X`.

A `Pointer` can be `nil` (not pointing to any entry), in which case it is represented by the all-zero 32-byte hash:
    
    0x0000000000000000000000000000000000000000000000000000000000000000

### Program

Program encapsulates the version of the [VM](vm1.md) and the bytecode that should be executed by that VM.

Field            | Type                | Description
-----------------|---------------------|----------------
VM Version       | [Integer](#integer) | The VM version to be used when evaluating the program.
Bytecode         | [String](#string)   | The program code to be executed.

#### Program Validation

**Inputs:**

1. program,
2. arguments (list of strings),
3. transaction version (integer).

**Algorithm:**

1. If the `VM Version` is greater than 1:
    1. If the transaction version is 1, validation fails.
    2. If the transaction version is greater than 1, validation succeeds.
2. If the `VM Version` is equal to 1:
    1. Instantiate [VM version 1](vm1.md) with initial state and expansion flag set to `true` iff transaction version is greater than 1.
    2. Evaluate the `Bytecode` with the given arguments.
    3. If the program evaluates successfully, validation succeeds. If the program fails evaluation, validation fails.


### Asset Definition

Field                 | Type                | Description
----------------------|---------------------|----------------
Initial Block ID      | [Hash](#hash)       | [ID](#entry-id) of the genesis block for the blockchain in which this asset is defined.
Issuance Program      | [Program](#program) | Program that must be satisfied for this asset to be issued.
Asset Reference Data  | [Hash](#hash)       | Hash of the reference data (formerly known as the “asset definition”) for this asset.


### Asset ID

Asset ID is a globally unique identifier of a given asset across all blockchains.

Asset ID is defined as the [SHA3-256](#sha3) of the [Asset Definition](#asset-definition):

    AssetID = SHA3-256(AssetDefinition)


### Asset Amount 1

AssetAmount1 struct encapsulates the number of units of an asset together with its [asset ID](#asset-id).

Field            | Type                 | Description
-----------------|----------------------|----------------
AssetID          | [Hash](#hash)        | [Asset ID](#asset-id).
Value            | [Integer](#integer)  | Number of units of the referenced asset.


### Value Source 1

An [Entry](#entry) uses a ValueSource to refer to other [Entries](#entry) that provide the value for it.

Field            | Type                        | Description
-----------------|-----------------------------|----------------
Ref              | [Pointer](#pointer)\<[Issuance1](#issuance-1)\|[Spend1](#spend-1)\|[Mux1](#mux-1)\> | Previous entry referenced by this ValueSource.
Value            | [AssetAmount1](#asset-amount-1) | Amount and Asset ID contained in the referenced entry.
Position         | [Integer](#integer)         | Iff this source refers to a [Mux](#mux-1) entry, then the `Position` is the index of an output. If this source refers to an [Issuance](#issuance-1) or [Spend](#spend-1) entry, then the `Position` must be 0.

#### Value Source 1 Validation

1. Verify that `Ref` is present and valid.
2. Define `RefDestination` as follows:
    1. If `Ref` is an [Issuance](#issuance-1) or [Spend](#spend-1):
        1. Verify that `Position` is 0.
        2. Define `RefDestination` as `Ref.Destination`.
    2. If `Ref` is a `Mux`:
        1. Verify that `Mux.Destinations` contains at least `Position + 1` ValueDestinations.
        2. Define `RefDestination` as `Mux.Destinations[Position]`.
3. Verify that `RefDestination.Ref` is equal to the ID of the current entry.
4. Verify that `RefDestination.Position` is equal to `SourcePosition`, where `SourcePosition` is defined as follows:
    1. If the current entry being validated is an [Output](#output-1) or [Retirement](#retirement-1), `SourcePosition` is 0.
    2. If the current entry being validated is a `Mux`, `SourcePosition` is the index of this `ValueSource` in the current entry's `Sources`.
5. Verify that `RefDestination.Value` is equal to `Value`.

### Value Destination 1

An Entry uses a ValueDestination to refer to other entries that receive value from the current Entry.

Field            | Type                           | Description
-----------------|--------------------------------|----------------
Ref              | [Pointer](#pointer)\<[Output1](#output-1)\|[Retirement1](#retirement-1)\|[Mux1](#mux-1)\> | Next entry referenced by this ValueDestination.
Value            | [AssetAmount1](#asset-amount-1)    | Amount and Asset ID contained in the referenced entry
Position         | [Integer](#integer)            | Iff this destination refers to a mux entry, then the Position is one of the mux's numbered Inputs. Otherwise, the position must be 0.

#### Value Destination 1 Validation

1. Verify that `Ref` is present. (This means it must be reachable by traversing `Results` and `Sources` starting from the TxHeader.)
2. Define `RefSource` as follows:
    1. If `Ref` is an `Output` or `Retirement`:
        1. Verify that `Position` is 0.
        2. Define `RefSource` as `Ref.Source`.
    2. If `Ref` is a `Mux`:
        1. Verify that `Ref.Sources` contains at least `Position + 1` ValueSources.
        2. Define `RefSource` as `Ref.Sources[Position]`.
3. Verify that `RefSource.Ref` is equal to the ID of the current entry.
4. Verify that `RefSource.Position` is equal to `DestinationPosition`, where `DestinationPosition` is defined as follows:
    1. If the current entry being validated is an `Issuance` or `Spend`, `DestinationPosition` is 0.
    2. If the current entry being validated is a `Mux`, `DestinationPosition` is the index of this `ValueDestination` in the current entry's `Destinations`.
5. Verify that `RefSource.Value` is equal to `Value`.


### Merkle Root

A top hash of a *merkle tree* (binary or patricia). Merkle roots are used within blocks to commit to a set of transactions and complete state of the blockchain. They are also used in merkleized programs and may also be used for structured reference data commitments.


### Merkle Binary Tree

The protocol uses a binary merkle hash tree for efficient proofs of validity. The construction is from [RFC 6962 Section 2.1](https://tools.ietf.org/html/rfc6962#section-2.1), but using SHA3–256 instead of SHA2–256. It is reproduced here, edited to update the hashing algorithm.

The input to the *merkle binary tree hash* (MBTH) is a list of data entries; these entries will be hashed to form the leaves of the merkle hash tree. The output is a single 32-byte hash value. Given an ordered list of n inputs, `D[n] = {d(0), d(1), ..., d(n-1)}`, the MBTH is thus defined as follows:

The hash of an empty list is the hash of an empty string:

    MBTH({}) = SHA3-256("")

The hash of a list with one entry (also known as a leaf hash) is:

    MBTH({d(0)}) = SHA3-256(0x00 || d(0))

For n > 1, let k be the largest power of two smaller than n (i.e., k < n ≤ 2k). The merkle binary tree hash of an n-element list `D[n]` is then defined recursively as

    MBTH(D[n]) = SHA3-256(0x01 || MBTH(D[0:k]) || MBTH(D[k:n]))

where `||` is concatenation and `D[k1:k2]` denotes the list `{d(k1), d(k1+1),..., d(k2-1)}` of length `(k2 - k1)`. (Note that the hash calculations for leaves and nodes differ. This domain separation is required to give second preimage resistance.)

Note that we do not require the length of the input list to be a power of two. The resulting merkle binary tree may thus not be balanced; however, its shape is uniquely determined by the number of leaves.

![Merkle Binary Tree](merkle-binary-tree.png)


### Merkle Patricia Tree

The protocol uses a binary radix tree with variable-length branches to implement a *merkle patricia tree*. This tree structure is used for efficient concurrent updates of the [assets merkle root](#assets-merkle-root) and compact recency proofs for unspent outputs.

The input to the *merkle patricia tree hash* (MPTH) is a list of key-value pairs of binary strings of arbitrary length ordered lexicographically by keys. Keys are unique bitstrings of a fixed length (length specified for each instance of the tree). Values are bitstrings of arbitrary length and are not required to be unique. Given a list of sorted key-value pairs, the MPTH is thus defined as follows:

The hash of an empty list is a 32-byte all-zero string:

    MPTH({}) = 0x0000000000000000000000000000000000000000000000000000000000000000

The hash of a list with one entry (also known as a leaf hash) is:

    MPTH({(key,value)}) = SHA3-256(0x00 || value)

In case a list contains multiple items, all keys have a common bit-prefix extracted and the list is split in two lists A and B with elements in each list sharing at least one prefix bit of their keys. This way the top level hash may have an empty common prefix, but nested hashes never have an empty prefix. The hash of multiple items is defined recursively:

    MPTH(A + B) = SHA3-256(0x01 || MPTH(A) || MPTH(B))

![Merkle Patricia Tree](merkle-patricia-tree.png)



### Transactions Merkle Root

Root hash of the [merkle binary hash tree](#merkle-binary-tree) formed by the [transaction IDs](#transaction-id) of all transactions included in the block.

### Assets Merkle Root

Root hash of the [merkle patricia tree](#merkle-patricia-tree) formed by [unspent outputs version 1](#output-1) after applying the [block](#block-header). Allows bootstrapping nodes from recent blocks and an archived copy of the corresponding merkle patricia tree without processing all historical transactions.

The tree contains unspent [outputs version 1](#output-1) (one or more per [asset ID](#asset-id)) where both key and value are the same value — the [Output ID](#output-1) of the unspent output.

Key                     | Value
------------------------|------------------------------
[Output ID](#output-1)  | [Output ID](#output-1)







## Entries

Entries form a _directed acyclic graph_ within a blockchain: [block headers](#block-header) reference the [transaction headers](#transaction-header) (organized in a [merkle tree](#merkle-binary-tree)) that in turn reference [outputs](#output-1), that are coming from [muxes](#mux-1), [issuances](#issuance-1) and [spends](#spend-1).

### Entry

Each entry has the following generic structure:

Field               | Type                 | Description
--------------------|----------------------|----------------------------------------------------------
Type                | String               | The type of this Entry. E.g. [Issuance](#issuance-1), [Retirement](#retirement-1) etc.
Body                | Struct               | Varies by type.
Witness             | Struct               | Varies by type.

### Entry ID

An entry’s ID is based on its _type_ and _body_. The type is encoded as raw sequence of bytes (without a length prefix).
The body is encoded as a SHA3-256 hash of all the fields of the body struct concatenated.

    entryID = SHA3-256("entryid:" || type || ":" || SHA3-256(body))


### Block Header

Field      | Type                 | Description
-----------|----------------------|----------------
Type       | String               | "blockheader"
Body       | Struct               | See below.  
Witness    | Struct               | See below.

Body field               | Type              | Description
-------------------------|-------------------|----------------------------------------------------------
Version                  | Integer                 | Block version, equals 1.
Height                   | Integer                 | Block serial number.
Previous Block ID        | Hash                    | [Hash](#block-id) of the previous block or all-zero string.
Timestamp                | Integer                 | Time of the block in milliseconds since 00:00:00 UTC Jan 1, 1970.
Transactions Merkle Root | MerkleRoot<TxHeader>    | Root hash of the [merkle binary hash tree](#merkle-binary-tree) formed by the transaction IDs of all transactions included in the block.
Assets Merkle Root       | PatriciaRoot<Output1>   | Root hash of the [merkle patricia tree](#merkle-patricia-tree) of the set of unspent outputs with asset version 1 after applying the block. See [Assets Merkle Root](#assets-merkle-root) for details.
Next Consensus Program Bytecode | String | Authentication predicate for adding a new block after this one.
ExtHash                  | [ExtStruct](#extension-struct)  | Extension fields.

Witness field            | Type              | Description
-------------------------|-------------------|----------------------------------------------------------
Program Arguments        | List\<String\>    | List of [signatures](#signature) and other data satisfying previous block’s next consensus program.

#### Block Header Validation

**Inputs:** 

1. BlockHeader entry,
2. BlockHeader entry from the previous block, `PrevBlockHeader`.
3. List of transactions included in block.

**Algorithm:**

1. Verify that the block’s version is greater or equal the block version in the previous block header.
2. Verify that `Height` is equal to `PrevBlockHeader.Height + 1`.
4. Verify that `PreviousBlockID` is equal to the entry ID of `PrevBlockHeader`.
5. Verify that `Timestamp` is greater than `PrevBlockHeader.Timestamp`.
6. Evaluate program `PreviousBlockID.NextConsensusProgram` with [VM version 1](vm1.md) and expansion flag set to `false`. If program execution failed, fail validation.
7. For each transaction in the block:
    1. [Validate transaction](#transaction-header-validation) with the timestamp and block version of the input block header.
8. Compute the [transactions merkle root](#transactions-merkle-root) for the block.
9. Verify that the computed merkle tree hash is equal to `TransactionsMerkleRoot`.
10. If the block version is 1: verify that the `ExtHash` is the all-zero hash.

### Block ID

Block ID is defined as an [Entry ID](#entry-id) of the [block header](#block-header) structure.

### Transaction Header

Field      | Type                 | Description
-----------|----------------------|----------------
Type       | String               | "txheader"
Body       | Struct               | See below.  
Witness    | Struct               | Empty struct.

Body Field | Type                                    | Description
-----------|-----------------------------------------|-------------------------
Version    | Integer                                 | Transaction version, equals 1.
Results    | List\<Pointer\<Output 1\|Retirement 1\>\>   | A list of pointers to Outputs or Retirements. This list must contain at least one item.
Data       | Hash                                    | Hash of the reference data for the transaction, or a string of 32 zero-bytes (representing no reference data).
Mintime    | Integer                                 | Must be either zero or a timestamp lower than the timestamp of the block that includes the transaction
Maxtime    | Integer                                 | Must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
ExtHash    | [ExtStruct](#extension-struct)                 | Hash of all extension fields. (See [Extstruct](#extension-struct).) If `Version` is known, this must be 32 zero-bytes.

### Transaction ID

Transaction ID is defined as an [Entry ID](#entry-id) of the [transaction header](#transaction-header) structure.


#### Transaction Header Validation

**Inputs:**

1. TxHeader entry,
2. timestamp,
3. block version.

**Algorithm:**

1. If the block version is 1, verify that version is equal to 1.
2. If the `Maxtime` is greater than zero, verify that it is greater than or equal to the `Mintime`.
3. If the `Mintime` is greater than zero:
    1. Verify that the input timestamp is greater than or equal to the `Mintime`.
4. If the transaction maxtime is greater than zero:
    1. Verify that the input timestamp is less than or equal to the `Maxtime`.
5. Check that `Results` includes at least one item.
6. Check that each of the `Results` is present and valid.
7. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.



### Output 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "output1"
Body                | Struct               | See below.
Witness             | Struct               | Empty struct.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource1         | The source of the units to be included in this output.
ControlProgram      | Program              | The program to control this output.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | [ExtStruct](#extension-struct) | If the transaction version is known, this must be 32 zero-bytes.


#### Output Validation

1. [Validate](#value-source-1-validation) `Source`.
2. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.


#### Retirement 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "retirement1"
Body                | Struct               | See below.
Witness             | Struct               | Empty struct.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource1         | The source of the units that are being retired.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | [ExtStruct](#extension-struct) | If the transaction version is known, this must be 32 zero-bytes.

#### Retirement Validation

1. [Validate](#value-source-1-validation) `Source`.
2. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.

### Spend 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "spend1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

Body field          | Type                 | Description
--------------------|----------------------|----------------
SpentOutput         | Pointer<Output1>      | The Output entry consumed by this spend.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | [ExtStruct](#extension-struct) | If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                 | Description
--------------------|----------------------|----------------
Destination         | ValueDestination1    | The Destination ("forward pointer") for the value contained in this spend. This can point directly to an Output entry, or to a Mux, which points to Output entries via its own Destinations.
Arguments           | List<String>         | Arguments for the control program contained in the SpentOutput.

#### Spend Validation

1. Verify that `SpentOutput` is present in the transaction, but do not validate it.
2. [Validate program](#program-validation) `SpentOutput.ControlProgram` with the given `Arguments` and the transaction version.
3. Verify that `SpentOutput.Value` is equal to `Destination.Value`.
4. [Validate](#value-destination-1-validation) `Destination`.
5. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.

### Issuance 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "issuance1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

Body field          | Type                   | Description
--------------------|------------------------|----------------
Anchor              | Pointer<Nonce1|Spend1> | Used to guarantee uniqueness of this entry.
Value               | AssetAmount1           | Asset ID and amount being issued.
Data                | Hash                   | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | [ExtStruct](#extension-struct)| If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                                      | Description
--------------------|-------------------------------------------|----------------
Destination         | ValueDestination1                         | The Destination ("forward pointer") for the value contained in this spend. This can point directly to an `Output`, or to a `Mux`, which points to `Output` entries via its own `Destinations`.
AssetDefinition     | [Asset Definition](#asset-definition)     | Asset definition for the asset being issued.
Arguments           | List<String>                              | Arguments for the control program contained in the SpentOutput.

#### Issuance Validation

**Inputs:**

1. Issuance entry,
2. initial block ID.

**Algorithm:**

1. Verify that `AssetDefinition.InitialBlockID` is equal to the given initial block ID.
2. Verify that the SHA3-256 hash of `AssetDefinition` is equal to `Value.AssetID`.
3. [Validate issuance program](#program-validation) `AssetDefinition.Program` with the given `Arguments` and the transaction version.
4. Verify that `Anchor` entry is present and is either [Nonce](#nonce) or [Spend](#spend-1) entry.
5. Validate the `Anchor` entry.
6. [Validate](#value-destination-1-validation) `Destination`.
7. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.


### Mux 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "mux1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Sources             | List<ValueSource1>   | The source of the units to be included in this Mux.
Program             | Program              | A program that controls the value in the Mux and must evaluate to true.
ExtHash             | [ExtStruct](#extension-struct) | If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                       | Description
--------------------|----------------------------|----------------
Destinations        | List<ValueDestination1>    | The Destinations ("forward pointers") for the value contained in this Mux. This can point directly to Output entries, or to other Muxes, which point to Output entries via their own Destinations.
Arguments           | String                     | Arguments for the program contained in the Nonce.

#### Mux Validation

1. [Validate](#program-validation) `Program` with the given `Arguments` and the transaction version.
2. For each `Source` in `Sources`, [validate](#value-source-1-validation) `Source`.
3. For each `Destination` in `Destinations`, [validate](#value-destination-1-validation) `Destination`.
4. For each `AssetID` represented in `Sources` and `Destinations`:
    1. Sum the total `Amounts` of the `Sources` with that asset ID. Validation fails if the sum overflows 63-bit integer.
    2. Sum the total `Amounts` of the `Destinations` with that asset ID. Validation fails if the sum overflows 63-bit integer.
    3. Verify that the two sums are equal.
5. Verify that for every asset ID among `Destinations`, there is at least one `Source` with such asset ID. (This prevents creating zero units of an asset not present among the valid sources.)
6. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.



### Nonce

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "nonce"
Body                | Struct               | See below.
Witness             | Struct               | See below.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Program             | Program              | A program that protects the nonce against replay and must evaluate to true.
Time Range          | Pointer<TimeRange>   | Reference to a TimeRange entry.
ExtHash             | [ExtStruct](#extension-struct) | If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                         | Description
--------------------|------------------------------|----------------
Arguments           | List<String>                 | Arguments for the program contained in the Nonce.
Issuance            | Pointer<Issuance1>            | Pointer to an issuance entry.

#### Nonce Validation

1. [Validate](#program-validation) `Program` with the given `Arguments`.
2. Verify that `Issuance` points to an issuance that is present in the transaction (meaning visitable by traversing `Results` and `Sources` from the transaction header) and whose `Anchor` is equal to this nonce's ID.
3. Verify that both mintime and maxtime in the `TimeRange` are not zero.
4. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.

### Time Range

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "timerange"
Body                | Struct               | See below.
Witness             | Struct               | Empty struct.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Mintime             | Integer              | Minimum time for this transaction.
Maxtime             | Integer              | Maximum time for this transaction.
ExtHash             | [ExtStruct](#extension-struct) | If the transaction version is known, this must be 32 zero-bytes.

#### Time Range Validation

1. Verify that `Mintime` is equal to or less than the `Mintime` specified in the [transaction header](#transaction-header).
2. Verify that `Maxtime` is either zero, or is equal to or greater than the `Maxtime` specified in the [transaction header](#transaction-header).
3. If the transaction version is 1: verify that the `ExtHash` is the all-zero hash.
