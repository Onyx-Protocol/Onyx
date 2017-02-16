### Transaction Serialization for Hashing

Serialization is defined using the serialization primitives [Byte](#byte), [Hash](#hash), [Integer](#integer), [String](#string), [List](#list) and [Struct](#struct). 

When hashing, a transaction is composed of a set of Transaction Entries, all of which inherit from the common type [Abstract Entry](#abstract-entry). 

Entries are used to define transactions. For example, they are used to specify inputs or outputs and add reference data. Each Transaction must include a [Header Entry](#header-entry). Entries can be identified by their [Entry ID](#entry-id). 

Older nodes may come across Entries with a type that they don't recognize. All unrecognized Entries should be treated as [Unknown Entries](#unknown-entry).

Field               | Type                 | Description
--------------------|----------------------|----------------------------------------------------------
Header              | Header               | A Header entry.
Entries             | List<AbstractEntry>  | A list of TransactionEntries. 


#### Byte

A `Byte` is encoded as 1 byte.

#### Hash

A `Hash` is encoded as 32 bytes. 

#### Integer

An `Integer` is encoded a [Varint63](#varint63).

#### String

A `String` is encoded as a [Varstring31](#varstring31).

#### List

A `List` is encoded as a [Varstring31](#varstring31) containing the serialized items, one by one, as defined by the schema. 

#### Struct

A `Struct` is encoded as a concatenation of all its serialized fields. 

#### Extstruct

An `ExtStruct` is encoded as a single 32-byte hash.

#### Hashable 

A Hashable is any type for which a hashing serialization is defined. 

#### Entry ID 

An entry's ID is based on its type and body. The body is length-prefixed, and its type is varint-encoded. 

```
entryID = HASH("entryid:" || entry.type || ":" || entry.body_hash)
```

#### Entry Pointer

A Pointer is encoded as a Hash, and identifies another entry by its ID. It also restricts the possible acceptable types: Pointer<X> must refer to an entry of type X. 

A Pointer can be `nil`, in which case it is represented by the all-zero 32-byte hash `0x00000000000000000000000000000000`.

#### ValueSource

An Entry uses a ValueSource to refer to other Entries that provide inputs to the initial Entry. 

Field            | Type                        | Description
-----------------|-----------------------------|----------------
Ref              | Pointer<Issuance|Spend|Mux> | Previous entry referenced by this ValueSource.
Value            | AssetAmount                 | Amount and Asset ID contained in the referenced entry. 
Position         | Integer                     | Iff this source refers to a mux entry, then the Position is one of the mux's numbered Outputs. If this source refers to an inp

#### ValueDestination 

An Entry uses a ValueDestination to refer to other entries that result from the initial Entry.

Field            | Type                           | Description
-----------------|--------------------------------|----------------
Ref              | Pointer<Output|Retirement|Mux> | Next entry referenced by this ValueSource.
Value            | AssetAmount                    | Amount and Asset ID contained in the referenced entry
Position         | Integer                        | Iff this destination refers to a mux entry, then the Position is one of the mux's numbered Inputs. Otherwise, the position must be 0.


#### Abstract Entry

Field               | Type                 | Description
--------------------|----------------------|----------------------------------------------------------
Type                | String               | The type of this Entry. e.g. Issuance, Retirement
Body                | Hashable             | Varies by type. 
Witness             | Hashable             | Varies by type. 

#### Unknown Entry

When older nodes come across an Entry with a type that they don't recognize, that Entry must be treated as an Unknown Entry. 

The type string must be transmitted explicitly from newer nodes to older news, so that older nodes can compute the Entry ID correctly.

Field                    | Type                 | Description
-------------------------|----------------------|----------------
Type                     | String               | Not statically known.
Body_Hash                | Hash                 | - 
Witness_Hash             | Hash                 | - 

#### TxHeader

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "txheader"
Body                | Hashable             | See below.  
Witness             | Hashable             | See below. 

##### TxHeader Body
 
Field      | Type                                          | Description
-----------|-----------------------------------------------|-------------------------
Version    | Integer                                       | Transaction version, equals 1.
Results    | List<Pointer<Output|Retirement|UnknownEntry>> | A list of pointers to "results." If the version is known, result entries must be Outputs or Retirements. This list must contain at least one item. 
Data       | Pointer<Data|UnknownEntry>                    | A single pointer to a Data or Unknown entry.
Mintime    | Integer                                       | Must be either zero or a timestamp higher than the timestamp of the block that includes the transaction
Maxtime    | Integer                                       | Must be either zero or a timestamp lower than the timestamp of the block that includes the transaction.
ExtHash    | Hash                                          | Hash of all struct extensions. (See [Extstruct](#extstruct).) If the version is known, all ext_hashes must be hashes of empty strings. 

##### TxHeader Witness

Field               | Type                 | Description
--------------------|----------------------|----------------
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Data

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "data1"
Body                | Hash                 | Hash of the underlying data. 
Witness             | Hash                 | Hash of empty string.

The body is a hash of the underlying data. The underlying data may not be known. If a transaction author wants to provide the underlying data, it must be done in the transport layer alongisde the actual transaction.

TKTK Address comments about specifying the hash function for the underlying data. I know we sorted this out, but now I can't remember.

#### Output 

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "output1"
Body                | Hashable             | See below. 
Witness             | Hashable             | See below.

##### Output Body

Field               | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource          | The source of the units to be included in this output.
ControlProgram      | Program              | The program to control this output.
Data                | Pointer<Data>        | Reference data included on this entry.
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

##### Output Witness

Field               | Type                 | Description
--------------------|----------------------|----------------
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Retirement 

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "retirement1"
Body                | Hashable             | See below. 
Witness             | Hashable             | See below.

##### Retirement Body

Field               | Type                 | Description
--------------------|----------------------|----------------
Source              | ValueSource          | The source of the units that are being retired.
Data                | Pointer<Data>        | Reference data included on this entry. 
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

##### Retirement Witness


Field               | Type                 | Description
--------------------|----------------------|----------------
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Spend  

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "spend1"
Body                | Hashable             | See below. 
Witness             | Hashable             | See below.

##### Spend Body

Field               | Type                 | Description
--------------------|----------------------|----------------
SpentOutput         | Pointer<Output>      | The Output Entry consumed by this spend.
Data                | Pointer<Data>        | Reference data included on this entry. 
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

##### Spend Witness

Field               | Type                 | Description
--------------------|----------------------|----------------
Destination         | ValueDestination     | The Destination ("forward pointer") for the value contained in this spend. This can point directly to an Output Entry, or to a Mux, which points to Output Entries via its own Destinations.
Arguments           | String               | Arguments for the control program contained in the SpentOutput. 
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

#### Issuance  

Field               | Type                 | Description
--------------------|----------------------|----------------
Type                | String               | "issuance1"
Body                | Hashable             | See below. 
Witness             | Hashable             | See below.

##### Issuance Body

Field               | Type                 | Description
--------------------|----------------------|----------------
Anchor              | Pointer<Nonce|Spend> | Used to guarantee uniqueness of this entry.
Value               | AssetAmount          | Asset ID and amount being issued. 
Data                | Pointer<Data>        | Reference data included on this entry.
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.

##### Issuance Witness

Field               | Type                 | Description
--------------------|----------------------|----------------
Anchor              | Pointer<Nonce|Spend> | Used to guarantee uniqueness of this entry.
Value               | AssetAmount          | Asset ID and amount being issued. 
Data                | Pointer<Data>        | Reference data included on this entry.
ExtHash             | Hash                 | If the transaction version is known, this must be the hash of the empty string.
