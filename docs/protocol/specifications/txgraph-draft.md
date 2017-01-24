# Transaction graph structure

## Transaction entry types

All entries inherit from a common type `AbstractEntry`.

    Header
    Issuance
    Input
    Output
    Retirement
    Mux
    Data
    Anchor
    Predicate
    TimeRange
    UnknownEntry = all other types, undefined in the current protocol version

### Entry ID

Identifier of the entry is based on its type and content. Content is length-prefixed, type is varint-encoded

    entry.id  = HASH(
        "entryid:"  || 
        entry.type  || ":" ||
        entry.content_hash || 
        concat(pointers.map{ entry.id })
    )

### Witness ID

Witness ID of the transaction is based on WIDs of intermediate entries.

    entry.wid = HASH(
        "entrywid:"        || 
        entry.id           || 
        entry.witness_hash ||
        concat(pointers.map{ entry.wid })
    )

## Abstract Entry (base type for all concrete entries)

    - type:      Integer
    - pointers:  List<Hash>
    - content:   ExtStruct
    - witness:   ExtStruct

Note: `pointers` is a flat list of hashes of other entries. 
The named fields for pointers and lists of pointers in the `content` are _indices_ to that flat list.

See [ExtStruct](#extstruct) description below.


## UnknownEntry

    - type:         unknown Integer
    - pointers:     List<Hash>
    - content.hash: Hash
    - witness.hash: Hash

## Header

    - type="header"
    - pointers: List<Hash>
    - content:
        version:       Integer
        results:       List<Pointer<Output|Retirement|UnknownEntry>>
        references:    List<Pointer<Data>>
        optional timerange:     TimeRange
        ext_hash:      Hash
    - witness:
        mintime:       Integer
        maxtime:       Integer
        ext_hash:      Hash

**Rules:**

1. If version is known, all `ext_hash`es must be hashes of empty strings, AbstractEntries are not allowed in pointers.
2. Results indices must correspond to pointers.
3. Results must contain at least one item.
4. Results must of type `Output|Retirement|UnknownEntry`.
5. Every result must be present and valid.
6. Every entry in `references` must be present and be of type `Data`.
7. mintime must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
8. maxtime must be either zero or a timestamp lower than the timestamp of the block that includes the transaction.


# Data

    - type="data1"
    - content_hash: Hash
    - witness_hash: HASH(empty_string)

**Rules:**

1. witness must be empty
2. Note: content itself may or may not be provided, but the content_hash must be known.


## Output

    - type="output1"
    - pointers:          List<Hash>
    - content:
        source:          Pointer<Mux>
        position:        Integer
        asset_id:        Hash
        amount:          Integer
        control_program: Program
        reference:       Pointer<Data>
        ext_hash:        Hash
    - witness:
        ext_hash:        Hash
   
**Rules:**

1. `amount` is in range.
2. `source` must be present and valid.
3. `source.destinations[position]` must equal self.id.
4. if tx version is known, all `ext_hash`es must be hashes of empty strings.
5. Insert `self.id` in utxo set. 

## Retirement

    - type="retirement1"
    - pointers:         List<Hash>
    - content:
        source:         Pointer<Mux>
        position:       Integer
        asset_id:       Hash
        amount:         Integer
        reference:      Pointer<Data>
        ext_hash:       Hash
    - witness:
        ext_hash:       Hash

**Rules:**

1. `amount` is in range
2. `source` must be present and valid
3. `source.destinations[position]` must equal self.id.
4. if tx version is known, all `ext_hash`es must be hashes of empty strings.

## Input

    - type="input1"
    - pointers:       List<Hash>
    - content:
        spent_output: Pointer<Output>
        reference:    Pointer<Data>
        ext_hash:     Hash
    - witness:
        destination:  Hash
        arguments:    String
        ext_hash:     Hash

**Rules:**

1. Validate that `spent_output` is present in tx.
2. The `spent_output.program` must evaluate to `true` with given `arguments`.
3. Remove `spent_output` from UTXO set.

NB: `spent_output` is not validated, as it was already validated in the transaction that added it to the UTXO.


## Issuance

    - type="issuance1"
    - pointers:           List<Hash>
    - content:
        anchor:           Pointer<Anchor|Input>
        asset_id:         Hash
        amount:           Integer
        reference:        Pointer<Data>
        ext_hash:         Hash
    - witness:
        destination:      Hash
        initial_block_id: Hash
        asset_definition: Pointer<Data>
        issuance_program: Program
        arguments:        String
        ext_hash:         Hash

**Rules:**

1. Check that `asset_id == HASH(initial_block_id, asset_definition, issuance_program)`.
2. The `issuance_program` must evaluate to `true` with given `arguments`.


## Anchor

    - type="anchor1"
    - pointers:     List<Hash>
    - content:
        program:    Program
        timerange:  TimeRange
        ext_hash:   Hash
    - witness:
        arguments:  String
        ext_hash:   Hash

**Rules:**

1. If tx version is known, the ext fields must be empty.
2. The ID of the anchor must be globally unique on the blockchain.
3. The `program` must evaluate to `true` with given `arguments`.


## Mux

    - type="mux1"
    - pointers:       List<Hash>
    - content:        
        sources:      List<Issuance|Input>
        ext_hash:     Hash
    - witness:
        destinations: List<Output|Retirement>
        ext_hash:     Hash

**Rules:**

1. For each source: `sources[i].destination` must equal `self.id` - prevents double-spending.
2. Each source must be unique in the `sources` list, no repetitions allowed.
3. Each identifier in `destinations` must be unique (no repetitions) and included in the transaction.
4. For each asset on the sources and destinations:
    1. Sum the source amounts of that asset (`sources[i].spent_output.amount`) and sum the destination amounts of that asset.
    2. Test that both source and destination sums are less than 2^63.
    3. Test that the source sum equals the destination sum.
    4. Check that there is at least one source with that asset ID.


## TimeRange

    - type="timerange"
    - pointers: empty list
    - content:
        mintime:  Integer
        maxtime:  Integer
        ext_hash: Hash
    - witness:
        ext_hash: Hash

**Rules:**

1. `mintime` must be equal to or greater than the `header.mintime` specified in the transaction header witness.
2. `maxtime` must be equal to or less than the `header.maxtime` specified in the transaction header witness.
    

## Program

This is not a separate entry, but an inlined struct that carries VM version and the program code in VM-specific encoding.

    Struct {
        vm_version: Integer
        code:       String
    }


## Serialization for hashing

Primitives:
    
    Byte
    Hash
    Integer
    String
    List
    Struct

* `Byte` is encoded as 1 byte.
* `Hash` is encoded as 32 bytes.
* `Integer` is encoded as LEB128, with 63-bit limit.
* `String` is encoded as LEB128 length prefix with 31-bit limit followed by raw bytes.
* `List` is encoded as LEB128 length prefix with 31-bit limit followed by serialized items, one by one (`Byte|Hash|Integer|String|List|Struct|ExtStruct`) as defined by the schema.
* `Struct` is encoded as concatenation of all its serialized fields.
* `ExtStruct` is encoded as concatenation of top fields with first `exthash` with a recursive rule for `exthash`. See below.

Common types:

    Pointer = Integer (an index in the entry.pointers field)
    Program = Struct{vm_version: Integer, code: String}



## ExtStruct

Extensible struct contains flat list of fields and the last field is the extension hash.

The remaining fields are committed to that extension hash. They are defined in flat namespace in a protobuf definition, but for the hashing purposes we group them recursively under another ExtStruct instance:

    ExtStruct {
        fields...
        exthash: Hash
        ext: ExtStruct
    }

    exthash == HASH(ext)
    HASH(ExtStruct) == HASH(serialized-fields || exthash)

Old clients ignore the extended fields (`ext`) and only see the first fields they understand plus the `exthash`.

If the extensibility is not allowed (e.g. when tx version is known), the last `exthash` must be a hash of an empty string.

**Examples:**

V1 schema:

    entry {
        a
        b
        exthash
    }
    ID = H(a || b || exthash1)

V2 schema:

    {
        a
        b
        exthash1
        c
        d
        exthash2
    }
    exthash1 = H(c || d || exthash2)
    ID = H(a || b || exthash1)

Note: when V2 data is encoded and sent to V1, V1 client drops `(c,d,exthash2)` fields and only uses `a,b,exthash1` fields to compute the ID. Which turns out the same as for V2 client.

V3 schema:

    {
        a
        b
        exthash1
        c
        d
        exthash2
        e
        f
        exthash3
    }  
    exthash2 = H(e || f || exthash3)
    exthash1 = H(c || d || exthash2)
    ID = H(a || b || exthash1)

The scheme is applied recursively for the subsequent updates.



## Translation Layer

### 1. OldTx -> NewTx

This is a first intermediate step that allows keeping old SDK, old tx index and data structures within Core, but refactoring how txs and outputs are hashed for UTXO set and merkle root in block headers.

    1. Create Header entry.
    2. If tx has non-zero mintime or maxtime, add TimeRange
    3. TBD

### 2. NewTx -> OldTx

This is a second intermediate step that allows keeping old SDK, but refactoring how txs are represented and stored internally in Core.

TODO: ...


## Compression

1. Serialization prefix indicates the format version that specifies how things are serialized and compressed.
2. Replace hashes with varint offsets, reconstruct hashes in real time and then verify that top hash matches the source (e.g. merkle tree item)
3. Replace some repeated elements such as initial block id with indices too.




