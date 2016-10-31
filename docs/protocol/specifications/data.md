# Data Model Specification

* [Introduction](#introduction)
* [Definitions](#definitions)
  * [Varint](#varint)
  * [Varint31](#varint31)
  * [Varint63](#varint63)
  * [Varstring31](#varstring31)
  * [Extensible string](#extensible-string)
  * [Public Key](#public-key)
  * [Signature](#signature)
  * [SHA3](#sha3)
  * [Optional Hash](#optional-hash)
  * [Block](#block)
  * [Block Serialization Flags](#block-serialization-flags)
  * [Block Header](#block-header)
  * [Block Commitment](#block-commitment)
  * [Block Witness](#block-witness)
  * [Block ID](#block-id)
  * [Block Signature Hash](#block-signature-hash)
  * [Transaction](#transaction)
  * [Transaction Entry](#transaction-entry)
  * [Reference Entry](#reference-entry)
  * [Issuance Entry](#issuance-entry)
  * [Input Entry](#input-entry)
  * [Output Entry](#output-entry)
  * [Retirement Entry](#retirement-entry)
  * [Issuance Hash](#issuance-hash)
  * [Outpoint](#outpoint)
  * [Transaction Serialization Flags](#transaction-serialization-flags)
  * [Transaction ID](#transaction-id)
  * [Transaction Witness Hash](#transaction-witness-hash)
  * [Transaction Signature Hash](#transaction-signature-hash)
  * [Program](#program)
  * [VM Version](#vm-version)
  * [Consensus Program](#consensus-program)
  * [Control Program](#control-program)
  * [Issuance Program](#issuance-program)
  * [Program Arguments](#program-arguments)
  * [Asset ID](#asset-id)
  * [Asset Definition](#asset-definition)
  * [Retired Asset](#retired-asset)
  * [Transactions Merkle Root](#transactions-merkle-root)
  * [Assets Merkle Root](#assets-merkle-root)
  * [Merkle Root](#merkle-root)
  * [Merkle Binary Tree](#merkle-binary-tree)
  * [Merkle Patricia Tree](#merkle-patricia-tree)
* [References](#references)


## Introduction

This document describes the serialization format for the blockchain data structures used in the Chain Protocol.

## Definitions

### Varint

[Little Endian Base 128](https://developers.google.com/protocol-buffers/docs/encoding) encoding for unsigned integers typically used to specify length prefixes for arrays and strings. Values in range [0, 127] are encoded in one byte. Larger values use two or more bytes.

### Varint31

A varint with a maximum allowed value of 0x7fffffff (2<sup>31</sup> – 1) and a minimum of 0. A varint31 fits into a signed 32-bit integer.

### Varint63

A varint with a maximum allowed value of 0x7fffffffffffffff (2<sup>63</sup> – 1) and a minimum of 0. A varint63 fits into a signed 64-bit integer.

### Varstring31

A binary string with a varint31 prefix specifying its length in bytes. So the empty string is encoded as a single byte 0x00, a one-byte string is encoded with two bytes 0x01 0xNN, a two-byte string is 0x02 0xNN 0xMM, etc. The maximum allowed length of the underlying string is 0x7fffffff (2<sup>31</sup> – 1).

### Extensible string

A varstring31 whose content is the concatenation of other encoded data structures, possibly including other varstring31s. Used for values that future versions of the protocol might wish to extend without breaking older clients. Older clients can consume the complete “outer” varstring31 and parse out the subparts they understand while ignoring the suffix that they don’t.

### Public Key

In this document, a *public key* is the 32-byte binary encoding
of an Ed25519 (EdDSA) public key, as defined in [CFRG1](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05).

### Signature

In this document, a *signature* is the 64-byte binary encoding
of an Ed25519 (EdDSA) signature, as defined in [CFRG1](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05).


### SHA3

*SHA3* refers to the SHA3-256 function as defined in [FIPS202](https://dx.doi.org/10.6028/NIST.FIPS.202) with a fixed-length 32-byte output.

This hash function is used throughout all data structures and algorithms in this spec,
with the exception of SHA-512 (see [FIPS180](http://csrc.nist.gov/publications/fips/fips180-2/fips180-2withchangenotice.pdf)) used internally as function H inside Ed25519 (see [CFRG1](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05)).

### Optional Hash

An *optional hash* is a function that transforms a variable-length binary string to another variable-length binary string using SHA3-256 with a special case for an empty string: it is hashed to an empty string.

Input String      | Optional Hash               | Varstring31-encoded Optional Hash
------------------|-----------------------------|----------------------------------------
Empty string      | Empty string                | One-byte string: 0x00
Non-empty string  | 32-byte result of SHA3-256  | 33-byte string; first byte equals 0x20



### Block

Field               | Type              | Description
--------------------|-------------------|----------------------------------------------------------
Serialization Flags | byte              | See [Block Serialization Flags](#block-serialization-flags).
Version             | varint63          | Block version, equals 1.
Height              | varint63          | Block serial number.
Previous Block ID   | sha3-256          | [Hash](#block-id) of the previous block or all-zero string.
Timestamp           | varint63          | Time of the block in milliseconds since 00:00:00 UTC Jan 1, 1970.
Block Commitment    | Extensible string | Extensible commitment string. See [Block Commitment](#block-commitment).
Block Witness       | Extensible string | Extensible witness string. See [Block Witness](#block-witness).
Transaction Count   | varint31          | Number of transactions that follow.
Transactions        | [Transaction]     | List of individual [transactions](#transaction).



### Block Serialization Flags

Serialization flags control what data is in a given block message and how it is encoded. Unused values are reserved for future expansion. Implementations must reject messages using unsupported serialization values.

The **first (least significant) bit** indicates whether the block includes [witness](#block-witness) data. If this bit is zero, the witness data is omitted entirely.

The **second bit** indicates whether the block includes [transaction](#transaction) data. If this bit is zero, the transaction count and transactions are omitted entirely.

Non-zero **higher bits** and value 0x02 are reserved for future use.

Serialization Flags Examples | Description
-----------------------------|---------------------------
0000 0000                    | Block with neither witness nor transactions. Used in [block signature hash](#block-signature-hash).
0000 0001                    | Block with witness but without transactions. Also called a “[block header](#block-header)”.
0000 0011                    | Block with both witness and transactions. Also called simply a “[block](#block)”.
0000 0010                    | Reserved for future use.



### Block Header

“Block header” is shorthand for a block serialized with [serialization flags](#block-serialization-flags) 0x01. Header does not contain actual transactions, but contains all commitments and witness data with block signatures.



### Block Commitment

The *block commitment* string allows extending blocks with additional data. For instance, a hypothetical future [VM version](#vm-version) might append a hash of an additional state available to the programming environment.

Unknown appended commitments must be ignored. Changes to the format of the commitment string may only append new fields, never remove or change the semantics of the existing ones.

Field                                   | Type        | Description
----------------------------------------|-------------|----------------------------------------------------------
Transactions Merkle Root                | sha3-256    | Root hash of the [merkle binary hash tree](#merkle-binary-tree) formed by the transaction witness hashes of all transactions included in the block.
Assets Merkle Root                      | sha3-256    | Root hash of the [merkle patricia tree](#merkle-patricia-tree) of the set of unspent [outputs](#output-entry) after applying the block. See [Assets Merkle Root](#assets-merkle-root) for details.
Next [Consensus Program](#consensus-program) | varstring31 | Authentication predicate for adding a new block after this one.
—                                       | —           | Additional fields may be added by future extensions.



### Block Witness

The *block witness* string contains cryptographic signatures and other data necessary for block verification. It allows extending blocks with additional witness data in future upgrades excluded from any signatures on this block, but committed to by the blocks that follow.

Witness Field           | Type          | Description
------------------------|---------------|----------------------------------------------------------
Program Arguments Count | varint31      | Number of [program arguments](#program-arguments) that follow.
Program Arguments       | [varstring31] | List of [signatures](#signature) and other data satisfying previous block’s [next consensus program](#consensus-program).
—                       | —             | Additional fields may be added by future extensions.

The entire witness data string (including any unsupported fields) is excluded from the [block’s signature hash](#block-signature-hash).


### Block ID

The *block ID* (also called *block hash*) is defined as [SHA3-256](#sha3) of the block serialized with 0x01 [serialization flags](#block-serialization-flags). This covers all header data including the block witness and the transactions’ merkle root, but it excludes transactions themselves.


### Block Signature Hash

The *block signature hash* is defined as [SHA3-256](#sha3) of the block serialized with 0x00 [serialization flags](#block-serialization-flags). This covers header data excluding the block witness, but including the [transactions’ merkle root](#transactions-merkle-root).

### Transaction

A *transaction* comprises the following fields concatenated:

Field               | Type          | Description
--------------------|---------------|----------------------------------------------------------
Serialization Flags | byte          | See [transaction serialization flags](#transaction-serialization-flags).
Version             | varint63      | Transaction version, equals 1.
Minimum Time        | varint63      | Zero or a block timestamp at which transaction becomes valid.
Maximum Time        | varint63      | Zero or a block timestamp after which transaction becomes invalid.
Entries Count       | varint31      | Number of transaction entries that follow.
Entries             | [TxEntry]     | List of [transaction entries](#transaction-entry).


### Transaction Entry

Transaction entry consists of three fields: *type*, *content* and *witness*. The type field is used to extend transactions with support of new entry formats in the future. Content and witness are extensible strings themselves that may be extended with additional fields in the future. Content is separated from the [entry witness](#entry-witness) in order to allow correct computation of [transaction ID](#transaction-id) and the [witness hash](#transaction-witness-hash) without need to parse and understand the content.

Field             | Type                | Description
------------------|---------------------|----------------------------------------------------------
Entry Type        | varint63            | [Type of the entry](#entry-type).
Entry Content     | Extensible string   | See [Entry Content](#entry-content).
Entry Witness     | Extensible string   | Optional [Entry Witness](#entry-witness) data. Absent if [serialization flags](#transaction-serialization-flags) do not have the witness bit set.


#### Entry Type

Entry type defines the structure and semantics for both the [content](#entry-content) and the [witness](#entry-witness) fields. 

The present version of Chain Protocol defines five entry types:

Entry type                             | Numeric value | Purpose
---------------------------------------|---------------|----------------
[Reference](#reference-entry)          | 0x00          | Provides transaction-level reference data without affecting asset flow. 
[Issuance](#issuance-entry)            | 0x01          | Creates new units of a given asset ID.
[Input](#input-entry)                  | 0x02          | Consumes existing units from a previous transaction’s output.
[Output](#output-entry)                | 0x03          | Distributes units to specified control program.
[Retirement](#retirement-entry)        | 0x04          | Removes units from circulation.

Other entry types are reserved for future extensions.

#### Entry Content

Exact format of the content field is defined according to the [entry type](#entry-type).

#### Entry Witness

The *witness* string contains [program arguments](#program-arguments) ([cryptographic signatures](#signature) and other data necessary to verify the input). Witness string does not affect the *outcome* of the transaction and therefore is excluded from the [transaction ID](#transaction-id).

The witness string can be extended with additional commitments, proofs or validation hints that are excluded from the [transaction ID](#transaction-id), but committed to the blockchain via the [witness hash](#transaction-witness-hash).

Exact format of the witness field is defined according to the [entry type](#entry-type).


### Reference Entry

Reference entries do not govern movement of assets, but only commit to an arbitrary transaction-level reference data (as opposed to entry-level reference data). Transactions may have zero or more such entries. Typically, every party to a transaction may add their own annotation to the entire transaction this way.

Entry content field stores an [optional hash](#optional-hash) of a reference data. Witness field is empty. Both content and witness fields can be extended in the future transaction versions.

#### Reference Entry Content

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Reference Data          | varstring31             | Arbitrary string or its [optional hash](#optional-hash), depending on [serialization flags](#transaction-serialization-flags).
—                       | —                       | Additional fields may be added by future extensions.

#### Reference Entry Witness

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
—                       | —                       | Additional fields may be added by future extensions.


### Issuance Entry

Unlike [inputs](#input-entry), each of which is unique because it references a distinct [output](#output-entry), issuances are not intrinsically unique and must be made so to protect against replay attacks. The field *nonce* contains an arbitrary string that must be distinct from the nonces in other issuances of the same asset ID during the interval between the transaction's minimum and maximum time. Nodes ensure uniqueness of the issuance by remembering the [issuance hash](#issuance-hash) that includes the nonce, asset ID and minimum and maximum timestamps. To make sure that *issuance memory* does not take an unbounded amount of RAM, network enforces the *maximum issuance window* for these timestamps.

If the transaction has another entry that guarantees uniqueness of the entire transaction (e.g. an [input entry](#input-entry)), then the issuance must be able to opt out of the bounded minimum and maximum timestamps and therefore the uniqueness test for the [issuance hash](#issuance-hash). The empty nonce signals if the input opts out of the uniqueness checks.

See [Validate Transaction](validation.md#validate-transaction) section for more details on how the network enforces the uniqueness of issuance inputs.

#### Issuance Entry Content

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Nonce                   | varstring31             | Variable-length string guaranteeing uniqueness of the issuing transaction or of the given issuance.
Asset ID                | sha3-256                | Global [asset identifier](#asset-id).
Amount                  | varint63                | Amount being issued.
Issuance Reference Data | varstring31             | Arbitrary string or its [optional hash](#optional-hash), depending on [serialization flags](#transaction-serialization-flags).
—                       | —                       | Additional fields may be added by future extensions.

#### Issuance Entry Witness

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Initial Block ID        | sha3-256                | Hash of the first block in this blockchain.
VM Version              | varint63                | [Version of the VM](#vm-version) that executes the issuance program.
Issuance Program        | varstring31             | Predicate defining the conditions of issue.
Program Arguments Count | varint31                | Number of [program arguments](#program-arguments) that follow.
Program Arguments       | [varstring31]           | [Signatures](#signature) and other data satisfying the issuance program. Used to initialize the [data stack](vm1.md#vm-state) of the VM.
—                       | —                       | Additional fields may be added by future extensions.

Note: nodes must verify that the initial block ID and issuance program are valid and match the declared asset ID in the corresponding content field.


### Input Entry

Input entries describe transfer of funds from a previous transaction’s [output](#output-entry)

#### Input Entry Content

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Outpoint Reference      | [Outpoint](#outpoint)   | [Transaction ID](#transaction-id) and index of the output being spent.
Input Reference Data    | varstring31             | Arbitrary string or its [optional hash](#optional-hash), depending on [serialization flags](#transaction-serialization-flags).
—                       | —                       | Additional fields may be added by future extensions.

#### Input Entry Witness

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Previous Output         | [Output Entry Content](#output-entry-content) | Output content field used as the source for this input [serialized with flags](#transaction-serialization-flags) 0x00 (this also means that reference data within this field is always encoded as hash, even if the transaction is serialized with direct reference data values in the entries).
Program Arguments Count | varint31                | Number of [program arguments](#program-arguments) that follow.
Program Arguments       | [varstring31]           | [Signatures](#signature) and other data satisfying the spent output’s control program. Used to initialize the [data stack](vm1.md#vm-state) of the VM.
—                       | —                       | Additional fields may be added by future extensions.


### Output Entry

Output entries describe allocation of assets using [control programs](#control-program).

#### Output Entry Content

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Asset ID                | sha3-256                | Global [asset identifier](#asset-id).
Amount                  | varint63                | Number of units of the specified asset.
VM Version              | varint63                | [Version of the VM](#vm-version) that executes the [control program](#control-program).
Control Program         | varstring31             | Predicate [program](#control-program) to control the specified amount.
Output Reference Data   | varstring31             | Arbitrary string or its [optional hash](#optional-hash), depending on [serialization flags](#transaction-serialization-flags).
—                       | —                       | Additional fields may be added by future extensions.

#### Output Entry Witness

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
—                       | —                       | Additional fields may be added by future extensions.



### Retirement Entry

Retirement entries remove units of assets from circulation. They are similar to [output entries](#output-entry), except they do not specify a control program since assets are made unspendable.

#### Retirement Entry Content

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Asset ID                | sha3-256                | Global [asset identifier](#asset-id).
Amount                  | varint63                | Number of units of the specified asset.
Retirement Reference Data   | varstring31         | Arbitrary string or its [optional hash](#optional-hash), depending on [serialization flags](#transaction-serialization-flags).
—                       | —                       | Additional fields may be added by future extensions.

#### Retirement Entry Witness

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
—                       | —                       | Additional fields may be added by future extensions.


### Issuance Hash

Issuance hash provides a globally unique identifier for an issuance input. It is defined as [SHA3-256](#sha3) of the following structure:

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Nonce                   | varstring31             | Nonce from the [issuance entry](#issuance-entry).
Asset ID                | sha3-256                | Global [asset identifier](#asset-id).
Minimum Time            | varint63                | Transaction minimum time.
Maximum Time            | varint63                | Transaction maximum time.

Note: the timestamp values are used exactly as specified in the [transaction](#transaction).


### Outpoint

An *outpoint* uniquely specifies a single transaction output.

Field                   | Type                    | Description
------------------------|-------------------------|----------------------------------------------------------
Transaction ID          | sha3-256                | [Transaction ID](#transaction-id) of the referenced transaction.
Output Index            | varint31                | Index (zero-based) of the [output entry](#output-entry) within the transaction.

Note: In the transaction wire format, outpoint uses the [varint encoding](#varint31) for the output index, but in the [assets merkle tree](#assets-merkle-root) a fixed-length big-endian encoding is used for lexicographic ordering of unspent outputs.


### Transaction Serialization Flags

Serialization flags control what and how data is encoded in a given [transaction](#transaction) message. Unused values are reserved for future expansion. Implementations must reject messages using unsupported serialization values. This allows changing encoding freely and extending the serialization flags fields to a longer sequence if needed.

The **first (least significant) bit** indicates whether the transaction includes witness data. If set to zero, the witness fields are absent from all [transaction entries](#transaction-entry).

The **second bit** indicates whether transaction reference data is present. If set to zero, the reference data is replaced by its optional hash value.

Both bits can be used independently. Non-zero **higher bits** are reserved for future use.

Serialization Flags Examples | Description
-----------------------------|---------------------------------------------------------------------------
0000 0000                    | Minimal serialization without witness and with reference data hashes instead of content.
0000 0001                    | Minimal serialization needed for full verification. Contains witness fields, but not complete reference data.
0000 0011                    | Full binary serialization with witness fields and reference data.


### Transaction ID

The *transaction ID* (also called *txid* or *transaction hash*) is defined as [SHA3-256](#sha3) of the transaction serialized with 0x00 [serialization flags](#transaction-serialization-flags). Thus, reference data is hashed via intermediate hashes and transaction witness data is excluded.

### Transaction Witness Hash

The *transaction witness hash* is defined as [SHA3-256](#sha3) of the following structure:

Field                           | Type                    | Description
--------------------------------|-------------------------|----------------------------------------------------------
Transaction ID                  | sha3-256                | [Transaction identifier](#transaction-id).
Entries Count                   | varint31                | Number of transaction entries.
Hashed Entry Witnesses          | [sha3-256]              | [SHA3-256](#sha3) hash of the [entry witness data](#entry-witness) from each entry (in the same order as entries).

Reusing the transaction ID saves time hashing again the rest of input and output data that can be arbitrarily large. Intermediate hashing of each witness enables compact proofs with partial reveal of witness data in cases of large transactions or large witnesses.

Note: hashes of input witness data cover not only the [program arguments](#program-arguments) with their count, but the rest of the data in the witness [extensible string](#extensible-string) (not including the length prefix of the entire witness string). So if the entry witness data contains two program arguments `0xaa` and `0xbbbb` and an additional suffix `0xffff`, then the witness data is defined as `0x0201aa02bbbbffff` (`0x02` being the number of arguments, `0x01` — the length of the first argument, `0x02` — the length of the second argument and additional data `0xffff` outside the scope of this specification). The hash of such data is then:

    SHA3-256(0x0201aa02bbbbffff) = 0x0c5ff1a162fc8ef7b742bfbf556c1a85ab404d27ec89e206c29f9a7e28b5f712


### Transaction Signature Hash

A *signature hash* (or *sighash*) corresponding to a given input is a hash of the [transaction ID](#transaction-id) and the index for that input. It is returned by [TXSIGHASH](vm1.md#txsighash), and is designed for use with [CHECKSIG](vm1.md#checksig) and [CHECKMULTISIG](vm1.md#checkmultisig) instructions.

The transaction signature hash is the [SHA3-256](#sha3) of the following structure:

Field                   | Type                                      | Description
------------------------|-------------------------------------------|----------------------------------------------------------
Transaction ID          | sha3-256                                  | Current [transaction ID](#transaction-id).
Input Index             | varint31                                  | Index of the current input encoded as [varint31](#varint31).
Output Commitment Hash  | sha3-256                                  | [SHA3-256](#sha3) of the output commitment from the output being spent by the current input. Issuance input uses a hash of an empty string.

Note 1. Including the spent output commitment makes it easier to verify the asset ID and amount at signing time, although those values are already committed to via the input's [outpoint](#outpoint).

Note 2. Using the hash of the output commitment instead of the output commitment as-is does not incur additional overhead since this hash is readily available from the [assets merkle tree](#assets-merkle-root). As a result, total amount of data to be hashed by all nodes during transaction validation is reduced.

### Program

A variable-length string of instructions executed by a virtual machine during blockchain verification.

### VM Version

A variable-length integer encoded as [varint63](#varint63) that defines bytecode format and virtual machine semantics for a program. See [VM Versioning](vm1.md#versioning) for more details.

### Consensus Program

The consensus program is a program declared in the [block commitment](#block-commitment) specifying a predicate for signing the next block after the current one.


### Control Program

The control program is a program specifying a predicate for transferring an asset; this is the asset’s destination. Control programs usually contain a hash of a contract allowing the actual contract code to be supplied later in the program arguments list of a spending transaction (along with authentication data such as digital signatures).

### Issuance Program

The issuance program is a [program](#program) specifying a predicate for issuing an asset within an [issuance entry](#issuance-entry). The asset ID is derived from the issuance program, guaranteeing the authenticity of the issuer.

Issuance programs must start with a [PUSHDATA](vm1.md#pushdata) opcode, followed by the [asset definition](#asset-definition), followed by a [DROP](vm1.md#drop) opcode.

### Program Arguments

A list of binary strings in the [issuance witness](#issuance-entry-witness), [input witness](#input-entry-witness) and [block witness](#block-witness) structures. It typically contains signatures and other data to satisfy the predicate specified by the control program of the output referenced by the current input. Program arguments are used also for authenticating *issuance inputs* where the predicate is defined by an issuance program.


### Asset ID

Globally unique identifier of a given asset. Future versions of the protocol may introduce new [issuance entry types](#issuance-entry) with a different definition of an asset ID, but an asset ID is always guaranteed to be unique across all blockchains.

Present version of the protocol defines asset ID as the [SHA3-256](#sha3) of the following structure:

Field            | Type          | Description
-----------------|---------------|-------------------------------------------------
Initial Block ID | sha3-256      | Hash of the first block in this blockchain.
VM Version       | varint63      | [Version of the VM](#vm-version) for the issuance program.
Issuance Program | varstring31   | Program used in the issuance input.


### Asset Definition

An asset definition is an arbitrary binary string that corresponds to a particular [asset ID](#asset-id). Future [issuance entry types](#issuance-entry) may define their own methods to declare and commit to asset definitions.

In he present version of the protocol, asset definitions are included in the issuance program. Issuance programs must start with a [PUSHDATA](vm1.md#pushdata) opcode, followed by the asset definition, followed by a [DROP](vm1.md#drop) opcode. Since the issuance program is part of the string hashed to determine an asset ID, the asset definition for a particular asset ID is immutable.

### Retired Asset

Units of an asset can be retired by allocating them to [retirement entry](#retirement-entry).

Retired assets are not included in the [assets merkle root](#assets-merkle-root) and therefore do not occupy any memory in the nodes. One may use a merkle path to the [transactions merkle root](#transactions-merkle-root) to create a compact proof for a retired asset.

Note: [outputs](#output-entry) with certain control programs may render the output unspendable (e.g. `FALSE` or `0 VERIFY`), but they do not cause the output to be removed from the [assets merkle root](#assets-merkle-root), only [retirement entries](#retirement-entry) do.

### Transactions Merkle Root

Root hash of the [merkle binary hash tree](#merkle-binary-tree) formed by the *transaction witness hashes* of all transactions included in the block.

### Assets Merkle Root

Root hash of the [merkle patricia tree](#merkle-patricia-tree) formed by unspent [output entries](#output-entry) after applying the block. Allows bootstrapping nodes from recent blocks and an archived copy of the corresponding merkle patricia tree without processing all historical transactions.

The tree contains unspent outputs (one or more per [asset ID](#asset-id)):

Key                       | Value
--------------------------|------------------------------
`<txhash><index int32be>` | [SHA3-256](#sha3) of the [output entry content](#output-entry-content)

Note: unspent output indices are encoded with a fixed-length big-endian format to support lexicographic ordering.

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









## References

* [FIPS180] [“Secure Hash Standard”, United States of America, National Institute of Standards and Technology, Federal Information Processing Standard (FIPS) 180-2](http://csrc.nist.gov/publications/fips/fips180-2/fips180-2withchangenotice.pdf).
* [FIPS202] [Federal Inf. Process. Stds. (NIST FIPS) - 202 (SHA3)](https://dx.doi.org/10.6028/NIST.FIPS.202)
* [LEB128] [Little-Endian Base-128 Encoding](https://developers.google.com/protocol-buffers/docs/encoding)
* [CFRG1] [Edwards-curve Digital Signature Algorithm (EdDSA) draft-irtf-cfrg-eddsa-05](https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05)
* [RFC 6962](https://tools.ietf.org/html/rfc6962#section-2.1)



