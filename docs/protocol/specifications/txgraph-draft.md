# Transaction graph structure

## Transaction entry types

All entries inherit from a common type `AbstractEntry`.

    Header
    Issuance
    Input
    Output
    Retirement
    Mux
    ReferenceData
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

    - type:      int
    - pointers: [Hash256]
    - content:   ExtensibleStruct
    - witness:   ExtensibleStruct

Note: `pointers` is a flat list of hashes of other entries. 
The named fields for pointers and lists of pointers in the `content` are _indices_ to that flat list.

See [ExtensibleStruct](#extensiblestruct) description below.


## UnknownEntry

    - type:         unknown int
    - pointers:    [Hash256]
    - content.hash: Hash256
    - witness.hash: Hash256

## Header

    - type=header
    - pointers: [Hash256]
    - content:
        version: int
        results: List<Output,
                      Retirement,
                      AbstractEntry>
        references: List<ReferenceData>
        timerange:  TimeRange
        ext_hash:   Hash256
    - witness:
        mintime
        maxtime
        ext_hash: Hash256

**Rules:**

1. If version is known, all `ext_hash`es must be hashes of empty strings, AbstractEntries are not allowed in pointers.
2. Results must contain at least one item.
3. Every result must be present and valid.
4. mintime must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
5. maxtime must be either zero or a timestamp lower than the timestamp of the block that includes the transaction.


# ReferenceData

    - type=refdata1
    - content: blob
    - witness: empty

**Rules:**

1. witness must be empty

## Output

    - type=output1
    - pointers: [Hash256]
    - content:
        source: Mux
        position: Integer
        reference_data: ReferenceData
        asset_id
        amount
        control_program: Program
        ext_hash: Hash256
    - witness:
        ext_hash: Hash256
   
**Rules:**

1. `amount` is in range.
2. `source` must be present and valid.
3. `source.destinations[position]` must equal self.id.
4. if tx version is known, all `ext_hash`es must be hashes of empty strings.
5. Insert `self.id` in utxo set. 

## Retirement

    - type=retirement1
    - pointers: [Hash256]
    - content:
        source: Mux
        position: Integer
        reference_data: ReferenceData
        asset_id
        amount
        ext_hash: Hash256
    - witness:
        ext_hash: Hash256

**Rules:**

1. `amount` is in range
2. `source` must be present and valid
3. `source.destinations[position]` must equal self.id.
4. if tx version is known, all `ext_hash`es must be hashes of empty strings.

## Input

    - type=input1
    - content:
        spent_output: Output
        ext_hash: Hash256
    - witness:
        destination: Hash256
        arguments: String
        ext_hash: Hash256

**Rules:**

1. Validate that `spent_output` is present in tx.
2. The `spent_output.program` must evaluate to `true` with given `arguments`.
3. Remove `spent_output` from UTXO set.

NB: `spent_output` is not validated, as it was already validated in the transaction that added it to the UTXO.


## Issuance

    - type=issuance1
    - content:
        anchor: Anchor|Input
        asset_id
        amount
        ext_hash: Hash256
    - witness:
        destination: Hash256
        initial_block_id
        asset_definition
        issuance_program: Program
        arguments: String
        ext_hash: Hash256

**Rules:**

1. Check that `asset_id == HASH(initial_block_id, asset_definition, issuance_program)`.
2. The `issuance_program` must evaluate to `true` with given `arguments`.


## Anchor

    - type=anchor1
    - content:
        program: Program
        timerange: TimeRange
        ext_hash: Hash256
    - witness:
        arguments: String
        ext_hash: Hash256

**Rules:**

1. If tx version is known, the ext fields must be empty.
2. The ID of the anchor must be globally unique on the blockchain.
3. The `program` must evaluate to `true` with given `arguments`.


## Mux

    - type=mux1
    - content:
        sources: List<Issuance|Input>
        ext_hash: Hash256
    - witness:
        destinations: List<Output|Retirement>
        ext_hash: Hash256

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

    - type=timerange
    - content:
        mintime: integer
        maxtime: integer
        ext_hash: Hash256
    - witness:
        ext_hash: Hash256

**Rules:**

1. `mintime` must be equal to or greater than the `header.mintime` specified in the transaction header.
2. `maxtime` must be equal to or less than the `header.maxtime` specified in the transaction header.
    

## Program

This is not a separate entry, but an inlined struct that carries VM version and the program code in VM-specific encoding.

    - vm_version: integer
    - code:       String


## ExtensibleStruct

Extensible struct contains flat list of fields and the last field is the extension hash.

The remaining fields are committed to that extension hash. They are defined in flat namespace in a protobuf definition, but for the hashing purposes we group them recursively under another ExtensibleStruct instance:
    
    ExtensibleStruct {
        fields...
        exthash: Hash256
        ext: ExtensibleStruct
    }
    
    exthash == Hash(ext)
    Hash(ExtensibleStruct) == Hash(serialized-fields || exthash)

Old clients ignore the extended fields (`ext`) and only see the first fields they understand plus the `exthash`.

If the extensibility is not allowed (e.g. when tx version is known), the last `exthash` must be a hash of an empty string.

### Examples:

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


## Serialization for hashing

Primitives:
    
    byte
    hash
    integer
    string
    list

* `byte` is encoded as 1 byte.
* `hash` is encoded as 32 bytes.
* `integer` is encoded as LEB128, with 63-bit limit.
* `string` is encoded as LEB128 length prefix with 31-bit limit followed by raw bytes.
* `list` is encoded as LEB128 length prefix with 31-bit limit followed by serialized items, one by one (`byte|hash|integer|string|list`) as defined by the schema.


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




