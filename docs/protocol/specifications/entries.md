# Entries Specification

* [Basic types](#basic-types)
  * [Hash](#hash)
  * [Integer](#integer)
  * [String](#string)
  * [List](#list)
  * [Struct](#struct)
  * [ExtStruct](#extstruct)
  * [Pointer](#pointer).
* [Auxiliary data structures](#auxiliary-data-structures)
  * [Asset Definition](#asset-definition)
  * [AssetAmount](#assetamount)
  * [Program](#program)
  * [ValueSource](#valuesource)
  * [ValueDestination](#valuedestination)
* [Entries](#entries)
  * [Entry](#entry)
  * [Entry ID](#entry-id)
  * [BlockHeader](#blockheader)
  * [TxHeader](#txheader)
  * [Output 1](#output-1)
  * [Retirement 1](#retirement-1)
  * [Spend 1](#spend-1)
  * [Issuance 1](#issuance-1)
  * [Nonce](#nonce)
  * [TimeRange](#timerange)
  * [Mux 1](#mux-1)

## Introduction

This is a specification of the semantic data structures used by blocks and transactions. These data structures and rules are used for validation and hashing. This format is independent from the format for transaction wire serialization.

A **block** is a set of [entries](#entries), which must include a [Block Header](#blockheader) entry.

A **transaction** is composed of a set of [entries](#entries). Each transaction must include a [TxHeader Entry](#txheader), which references other entries in the transaction, which in turn can reference additional entries. 

Every entry is identified by its [Entry ID](#entry-id).

## Basic types

All entries and [auxiliary data structures](#auxiliary-data-structures) are defined in terms of the following basic types:

* [Hash](#hash)
* [Integer](#integer)
* [String](#string)
* [List](#list)
* [Struct](#struct)
* [ExtStruct](#extstruct)
* [Pointer](#pointer).

### Hash

A `Hash` is encoded as 32 bytes.

### Integer

An `Integer` is encoded a [Varint63](data.md#varint63).

### String

A `String` is encoded as a [Varstring31](data.md#varstring31).

### List

A `List` is encoded as a [Varstring31](data.md#varstring31) containing the serialized items, one by one, as defined by the schema. 

Note: since the `List` is encoded as a variable-length string, its length prefix indicates not the number of _items_,
but the number of _bytes_ of all the items in their serialized form.

### Struct

A `Struct` is encoded as a concatenation of all its serialized fields.

### ExtStruct

An `ExtStruct` is encoded as a single 32-byte hash. 
Future versions of the protocol may add additional fields as “Extension Structs” that will be compressed in a single hash for backwards compatibility.

### Pointer

A `Pointer` is encoded as a [Hash](#hash), and identifies another [entry](#entry) by its [ID](#entry-id). 

`Pointer` restricts the possible acceptable types: `Pointer<X>` must refer to an entry of type `X`.

A `Pointer` can be `nil` (not pointing to any entry), in which case it is represented by the all-zero 32-byte hash:
    
    0x0000000000000000000000000000000000000000000000000000000000000000



## Auxiliary data structures

Auxiliary data structures are [Structs](#struct) that are not [entries](#entries) by themselves, but used as fields within the entries.

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
3. expansion flag (true/false),
4. transaction version (integer).

**Algorithm:**

1. If the `VM Version` is greater than 1:
    1. If the transaction version is 1, validation fails.
    2. If the transaction version is greater than 1, validation succeeds.
2. If the `VM Version` is equal to 1:
    1. Evaluate the `Bytecode` with the given arguments and a given expansion flag using [VM Version 1](vm1.md).
    2. If the program evaluates successfully, validation succeeds. If the program fails evaluation, validation fails.


### AssetDefinition

Field                 | Type                | Description
----------------------|---------------------|----------------
Initial Block ID      | [Hash](#hash)       | [ID](#entry-id) of the genesis block for the blockchain in which this asset is defined.
Issuance Program      | [Program](#program) | Program that must be satisfied for this asset to be issued.
Asset Reference Data  | [Hash](#hash)       | Hash of the reference data (formerly known as the “asset definition”) for this asset.


### Asset ID

Asset ID is a globally unique identifier of a given asset across all blockchains.

Asset ID is defined as the [SHA3-256](#sha3) of the [Asset Definition](#assetdefinition):

    AssetID = SHA3-256(AssetDefinition)


### AssetAmount

AssetAmount struct encapsulates the number of units of an asset together with its [asset ID](#asset-id).

Field            | Type                 | Description
-----------------|----------------------|----------------
AssetID          | [Hash](#hash)        | [Asset ID](#asset-id).
Value            | [Integer](#integer)  | Number of units of the referenced asset.


### ValueSource

An [Entry](#entry) uses a ValueSource to refer to other [Entries](#entry) that provide the value for it.

Field            | Type                        | Description
-----------------|-----------------------------|----------------
Ref              | [Pointer](#pointer)\<[Issuance](#issuance)\|[Spend](#spend)\|[Mux](#mux)\> | Previous entry referenced by this ValueSource.
Value            | [AssetAmount](#assetamount) | Amount and Asset ID contained in the referenced entry.
Position         | [Integer](#integer)         | Iff this source refers to a [Mux](#mux) entry, then the `Position` is the index of an output. If this source refers to an [Issuance](#issuance) or [Spend](#spend) entry, then the `Position` must be 0.

#### ValueSource Validation

1. Verify that `Ref` is present and valid.
2. Define `RefDestination` as follows:
    1. If `Ref` is an [Issuance](#issuance) or [Spend](#spend):
        1. Verify that `Position` is 0.
        2. Define `RefDestination` as `Ref.Destination`.
    2. If `Ref` is a `Mux`:
        1. Verify that `Mux.Destinations` contains at least `Position + 1` ValueDestinations.
        2. Define `RefDestination` as `Mux.Destinations[Position]`.
3. Verify that `RefDestination.Ref` is equal to the ID of the current entry.
4. Verify that `RefDestination.Position` is equal to `SourcePosition`, where `SourcePosition` is defined as follows:
    1. If the current entry being validated is an [Output](#output) or [Retirement](#retirement), `SourcePosition` is 0.
    2. If the current entry being validated is a `Mux`, `SourcePosition` is the index of this `ValueSource` in the current entry's `Sources`.
5. Verify that `RefDestination.Value` is equal to `Value`.

### ValueDestination

An Entry uses a ValueDestination to refer to other entries that receive value from the current Entry.

Field            | Type                           | Description
-----------------|--------------------------------|----------------
Ref              | [Pointer](#pointer)\<[Output](#output)\|[Retirement](#retirement)\|[Mux](#mux)\> | Next entry referenced by this ValueDestination.
Value            | [AssetAmount](#assetamount)    | Amount and Asset ID contained in the referenced entry
Position         | [Integer](#integer)            | Iff this destination refers to a mux entry, then the Position is one of the mux's numbered Inputs. Otherwise, the position must be 0.

#### ValueDestination Validation

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


#### Value flow validation

**Inputs:**

1. [ValueSource](#valuesource),
2. receiving entry (which contains [ValueSource](#valuesource) struct),
3. source index (within the receiving entry),
4. [ValueDestination](#valuedestination),
5. sending entry (which contains [ValueDestination](#valuedestination) struct),
6. destination index (within the receiving entry).

**Algorithm:**

1. Verify that `Value` fields in both source and destination are equal.
2. Verify that `ValueSource.Ref` is equal to the sending entry’s ID.
3. Verify that `ValueDestination.Ref` is equal to the receiving entry’s ID.
4. Verify that `ValueSource.Position` is equal to the destination index.
5. Verify that `ValueDestination.Position` is equal to the source index.


## Entries

Entries form a _directed acyclic graph_ within a blockchain: [block headers](#blockheader) reference the [transaction headers](#txheader) (organized in a [merkle tree](data.md#merkle-binary-tree)) that in turn reference [outputs](#output), that are coming from [muxes](#mux), [issuances](#issuance) and [spends](#spend).

### Entry

Each entry has the following generic structure:

Field               | Type                 | Description
--------------------|----------------------|----------------------------------------------------------
Type                | String               | The type of this Entry. E.g. [Issuance](#issuance), [Retirement](#retirement) etc.
Body                | Struct               | Varies by type.
Witness             | Struct               | Varies by type.

### Entry ID

An entry’s ID is based on its _type_ and _body_. The type is encoded as raw sequence of bytes (without a length prefix).
The body is encoded as a SHA3-256 hash of all the fields of the body struct concatenated.

    entryID = SHA3-256("entryid:" || type || ":" || SHA3-256(body))


### BlockHeader

Field      | Type                 | Description
-----------|----------------------|----------------
Type       | String               | "blockheader"
Body       | Struct               | See below.  
Witness    | Struct               | See below.

Body field               | Type              | Description
-------------------------|-------------------|----------------------------------------------------------
Version                  | Integer           | Block version, equals 1.
Height                   | Integer           | Block serial number.
Previous Block ID        | Hash              | [Hash](#block-id) of the previous block or all-zero string.
Timestamp                | Integer           | Time of the block in milliseconds since 00:00:00 UTC Jan 1, 1970.
Transactions Merkle Root | Hash    | Root hash of the [merkle binary hash tree](data.md#merkle-binary-tree) formed by the transaction IDs of all transactions included in the block.
Assets Merkle Root       | Hash    | Root hash of the [merkle patricia tree](data.md#merkle-patricia-tree) of the set of unspent outputs with asset version 1 after applying the block. See [Assets Merkle Root](data.md#assets-merkle-root) for details.
Next [Consensus Program](data.md#consensus-program) | String | Authentication predicate for adding a new block after this one.
ExtHash                  | [ExtStruct](#extstruct)    | Extension fields.

Witness field            | Type              | Description
-------------------------|-------------------|----------------------------------------------------------
Program Arguments        | List\<String\>    | List of [signatures](data.md#signature) and other data satisfying previous block’s [next consensus program](data.md#consensus-program).

#### BlockHeader Validation

**Inputs:** 

1. BlockHeader entry,
2. BlockHeader entry from the previous block, `PrevBlockHeader`.
3. List of transactions included in block.

**Algorithm:**

1. Verify that the block’s version is greater or equal the block version in the previous block header.
2. Verify that `Height` is equal to `PrevBlockHeader.Height + 1`. If not, halt and return false.
4. Verify that `PreviousBlockID` is equal to the entry ID of `PrevBlockHeader`.
5. Verify that `Timestamp` is greater than `PrevBlockHeader.Timestamp`.
6. Verify that `PreviousBlockID.NextConsensusProgram` with VM version 1.
7. For each transaction in the block:
    1. [Validate transaction](#validate-transaction) with the timestamp and block version of the input block header; if it is not valid, halt and return false.
8. Compute the [transactions merkle root](data.md#transactions-merkle-root) for the block.
9. Verify that the computed merkle tree hash is equal to `TransactionsMerkleRoot`.

### TxHeader

Field      | Type                 | Description
-----------|----------------------|----------------
Type       | String               | "txheader"
Body       | Struct               | See below.  
Witness    | Struct               | Empty struct.

Body Field | Type                                    | Description
-----------|-----------------------------------------|-------------------------
Version    | Integer                                 | Transaction version, equals 1.
Results    | List\<Pointer\<Output\|Retirement\>\>   | A list of pointers to Outputs or Retirements. This list must contain at least one item.
Data       | Hash                                    | Hash of the reference data for the transaction, or a string of 32 zero-bytes (representing no reference data).
Mintime    | Integer                                 | Must be either zero or a timestamp lower than the timestamp of the block that includes the transaction
Maxtime    | Integer                                 | Must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
ExtHash    | Hash                                    | Hash of all extension fields. (See [Extstruct](#extstruct).) If `Version` is known, this must be 32 zero-bytes.


#### TxHeader Validation

1. Check that `Results` includes at least one item.
2. Check that each of the `Results` is present and valid.


### Output 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "output1"
Body                | Struct               | See below.
Witness             | Struct               | Empty struct.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource          | The source of the units to be included in this output.
ControlProgram      | Program              | The program to control this output.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be 32 zero-bytes.


#### Output Validation

1. [Validate](#valuesource-validation) `Source`.


#### Retirement 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "retirement1"
Body                | Struct               | See below.
Witness             | Struct               | Empty struct.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource          | The source of the units that are being retired.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be 32 zero-bytes.

#### Retirement Validation

1. [Validate](#valuesource-validation) `Source`.

### Spend 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "spend1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

Body field          | Type                 | Description
--------------------|----------------------|----------------
SpentOutput         | Pointer<Output>      | The Output entry consumed by this spend.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                 | Description
--------------------|----------------------|----------------
Destination         | ValueDestination     | The Destination ("forward pointer") for the value contained in this spend. This can point directly to an Output entry, or to a Mux, which points to Output entries via its own Destinations.
Arguments           | List<String>         | Arguments for the control program contained in the SpentOutput.

#### Spend Validation

1. Verify that `SpentOutput` is present in the transaction (do not check that it is valid.)
2. [Validate](#program-validation) `SpentOutput.ControlProgram` with the given `Arguments`.
3. Verify that `SpentOutput.Value` is equal to `Destination.Value`.
4. [Validate](#valuedestination-validation) `Destination`.


### Issuance 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "issuance1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Anchor              | Pointer<Nonce|Spend> | Used to guarantee uniqueness of this entry.
Value               | AssetAmount          | Asset ID and amount being issued.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                                      | Description
--------------------|-------------------------------------------|----------------
Destination         | ValueDestination                          | The Destination ("forward pointer") for the value contained in this spend. This can point directly to an Output Entry, or to a Mux, which points to Output Entries via its own Destinations.
AssetDefinition     | [Asset Definition](#asset-definition)     | Asset definition for the asset being issued.
Arguments           | List<String>                              | Arguments for the control program contained in the SpentOutput.

#### Issuance Validation

1. Verify that the SHA3-256 hash of `AssetDefinition` is equal to `Value.AssetID`.
2. [Validate](#program-validation) `AssetDefinition.Program` with the given `Arguments`.
3. Verify that `Anchor` is present and valid.
4. [Validate](#valuedestination-validation) `Destination`.

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
ExtHash             | Hash                 | If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                         | Description
--------------------|------------------------------|----------------
Arguments           | List<String>                 | Arguments for the program contained in the Nonce.
Issuance            | Pointer<Issuance>            | Pointer to an issuance entry.

#### Nonce Validation

1. [Validate](#program-validation) `Program` with the given `Arguments`.
2. Verify that `Issuance` points to an issuance that is present in the transaction (meaning visitable by traversing `Results` and `Sources` from the transaction header) and whose `Anchor` is equal to this nonce's ID.


### TimeRange  

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "timerange"
Body                | Struct               | See below.
Witness             | Struct               | Empty struct.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Mintime             | Integer              | Minimum time for this transaction.
Maxtime             | Integer              | Maximum time for this transaction.
ExtHash             | Hash                 | If the transaction version is known, this must be 32 zero-bytes.

#### TimeRange Validation

1. Verify that `Mintime` is equal to or less than the `Mintime` specified in the [transaction header](#txheader).
2. Verify that `Maxtime` is either zero, or is equal to or greater than the `Maxtime` specified in the [transaction header](#txheader).

### Mux 1

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "mux1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

Body field          | Type                 | Description
--------------------|----------------------|----------------
Sources             | List<ValueSource>    | The source of the units to be included in this Mux.
Program             | Program              | A program that controls the value in the Mux and must evaluate to true.
ExtHash             | Hash                 | If the transaction version is known, this must be 32 zero-bytes.

Witness field       | Type                       | Description
--------------------|----------------------------|----------------
Destinations        | List<ValueDestination>     | The Destinations ("forward pointers") for the value contained in this Mux. This can point directly to Output entries, or to other Muxes, which point to Output entries via their own Destinations.
Arguments           | String                     | Arguments for the program contained in the Nonce.

#### Mux Validation

1. [Validate](#program-validation) `Program` with the given `Arguments`.
2. For each `Source` in `Sources`, [validate](#valuesource-validation) `Source`.
3. For each `Destination` in `Destinations`, [validate](#valuedestination-validation) `Destination`.
4. For each `AssetID` represented in `Sources` and `Destinations`:
    1. Sum the total `Amounts` of the `Sources` with that asset ID. Validation fails if the sum overflows 63-bit integer.
    2. Sum the total `Amounts` of the `Destinations` with that asset ID. Validation fails if the sum overflows 63-bit integer.
    3. Verify that the two sums are equal.
