# Entries Specification

TBD: Table of Contents

This is a specification of the semantic data structures used by transactions. These data structures and rules are used for validation and hashing. This format is independent from the format for [transaction wire serialization](data.md#transaction-wire-serialization).

A transaction is composed of a set of [transaction entries](#entries). Each transaction must include a [Header Entry](#header-entry), which references other entries in the transaction, which in turn can reference additional entries. Entries can be identified by their [Entry ID](#entry-id).

## Entry Serialization

An entry's ID is based on its type and body. The body is encoded as a [string](#string), and its type is [Varint63](data.md#varint63)-encoded.

```
entryID = HASH("entryid:" || entry.type || ":" || entry.body_hash)
```

## Fields

Each entry contains a defined combination of the following fields: [Byte](#byte), [Hash](#hash), [Integer](#integer), [String](#string), [List](#list), [Struct](#struct), [Extstruct](#extstruct), and [Entry Pointer](#entry-pointer).

Below is the serialization of those fields.

### Byte

A `Byte` is encoded as 1 byte.

### Hash

A `Hash` is encoded as 32 bytes.

### Integer

An `Integer` is encoded a [Varint63](data.md#varint63).

### String

A `String` is encoded as a [Varstring31](data.md#varstring31).

### List

A `List` is encoded as a [Varstring31](data.md#varstring31) containing the serialized items, one by one, as defined by the schema.

### Struct

A `Struct` is encoded as a concatenation of all its serialized fields.

### Extstruct

An `ExtStruct` is encoded as a single 32-byte hash.

### Entry Pointer

A Pointer is encoded as a Hash, and identifies another entry by its ID. It also restricts the possible acceptable types: Pointer<X> must refer to an entry of type X.

A Pointer can be `nil`, in which case it is represented by the all-zero 32-byte hash `0x00000000000000000000000000000000`.

## Data Structures

### ValueSource

An Entry uses a ValueSource to refer to other Entries that provide inputs to the initial Entry.

Field            | Type                        | Description
-----------------|-----------------------------|----------------
Ref              | Pointer<Issuance|Spend|Mux> | Previous entry referenced by this ValueSource.
Value            | AssetAmount                 | Amount and Asset ID contained in the referenced entry.
Position         | Integer                     | Iff this source refers to a mux entry, then the Position is one of the mux's numbered Outputs. If this source refers to an inp

### ValueDestination

An Entry uses a ValueDestination to refer to other entries that result from the initial Entry.

Field            | Type                           | Description
-----------------|--------------------------------|----------------
Ref              | Pointer<Output|Retirement|Mux> | Next entry referenced by this ValueSource.
Value            | AssetAmount                    | Amount and Asset ID contained in the referenced entry
Position         | Integer                        | Iff this destination refers to a mux entry, then the Position is one of the mux's numbered Inputs. Otherwise, the position must be 0.

## Entries

All entries have the following structure:

Field               | Type                 | Description
--------------------|----------------------|----------------------------------------------------------
Type                | String               | The type of this Entry. e.g. Issuance, Retirement
Body                | Struct               | Varies by type.
Witness             | Struct               | Varies by type.


### TxHeader

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "txheader"
Body                | Struct               | See below.  
Witness             | Struct               | See below.

#### TxHeader Body

Field      | Type                                          | Description
-----------|-----------------------------------------------|-------------------------
Version    | Integer                                       | Transaction version, equals 1.
Results    | List<Pointer<Output|Retirement>>              | A list of pointers to Outputs or Retirements. This list must contain at least one item.
Data       | Hash                                          | Hash of the reference data for the transaction, or a string of 32 zero-bytes (representing no reference data).
Mintime    | Integer                                       | Must be either zero or a timestamp lower than the timestamp of the block that includes the transaction
Maxtime    | Integer                                       | Must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
ExtHash    | Hash                                          | Hash of all struct extensions. (See [Extstruct](#extstruct).) If the version is known, all ext_hashes must be hashes of empty strings.

#### TxHeader Witness

Field               | Type                 | Description
--------------------|----------------------|----------------


### Output

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "output1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

#### Output Body

Field               | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource          | The source of the units to be included in this output.
ControlProgram      | Program              | The program to control this output.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Output Witness

Field               | Type                 | Description
--------------------|----------------------|----------------


#### Retirement

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "retirement1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

#### Retirement Body

Field               | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource          | The source of the units that are being retired.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Retirement Witness

Field               | Type                 | Description
--------------------|----------------------|----------------


### Spend  

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "spend1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

#### Spend Body

Field               | Type                 | Description
--------------------|----------------------|----------------
SpentOutput         | Pointer<Output>      | The Output Entry consumed by this spend.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Spend Witness

Field               | Type                 | Description
--------------------|----------------------|----------------
Destination         | ValueDestination     | The Destination ("forward pointer") for the value contained in this spend. This can point directly to an Output Entry, or to a Mux, which points to Output Entries via its own Destinations.
Arguments           | String               | Arguments for the control program contained in the SpentOutput.

### Issuance  

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "issuance1"
Body                | Struct               | See below.
Witness             | Struct               | See below.

#### Issuance Body

Field               | Type                 | Description
--------------------|----------------------|----------------
Anchor              | Pointer<Nonce|Spend> | Used to guarantee uniqueness of this entry.
Value               | AssetAmount          | Asset ID and amount being issued.
Data                | Hash                 | Hash of the reference data for this entry, or a string of 32 zero-bytes (representing no reference data).
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Issuance Witness

Field               | Type                 | Description
--------------------|----------------------|----------------
Anchor              | Pointer<Nonce|Spend> | Used to guarantee uniqueness of this entry.
Value               | AssetAmount          | Asset ID and amount being issued.
