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

    entry.id  = HASH("entryid:" || entry.type  || ":" || entry.content_hash)


## Abstract Entry (base type for all concrete entries)

    - type:      Integer
    - content:   ExtStruct
    - witness:   ExtStruct

See [ExtStruct](#extstruct) description below.


## UnknownEntry

    - type:         unknown Integer
    - content.hash: Hash
    - witness.hash: Hash

## Header

    entry {
        type="header"
        content:
            version:       Integer
            results:       List<Pointer<Output|Retirement|UnknownEntry>>
            references:    List<Pointer<Data>>
            mintime:       Integer
            maxtime:       Integer
            ext_hash:      Hash
        witness:
            ext_hash:      Hash        
    }

**Rules:**

1. If version is known, all `ext_hash`es must be hashes of empty strings, `UnknownEntry`s are not allowed in `results`.
2. Results must contain at least one item.
3. Results must of type `Output|Retirement|UnknownEntry`.
4. Every result must be present and valid.
6. Every entry in `references` must be present and be of type `Data`.
7. mintime must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
8. maxtime must be either zero or a timestamp lower than the timestamp of the block that includes the transaction.


# Data

    entry {
        type="data1"
        content_hash: Hash
        witness_hash: HASH(empty_string)
    }

**Rules:**

1. witness must be empty
2. Note: content itself may or may not be provided, but the content_hash must be known.


## Output

    entry {
        type="output1"
        content:
            source:          ValueSource
            control_program: Program
            reference:       Pointer<Data>
            ext_hash:        Hash
        witness:
            ext_hash:        Hash
    }

**Rules:**

1. Let `preventry` be the `source.ref`.
2. `preventry` must be present and valid.
3. Validate `source.amount` is in range.
4. Need to ensure that this output exclusively consumes the `preventry`'s destination.
5. If `preventry` is Input or Issuance:
    1. Let `dest` be `preventry.destination`.
6. If `preventry` is Mux:
    1. Let `dest` be `preventry.destinations[source.position]`. Fail if not found.
7. Verify previous entry's destination `dest`:
    1. Verify `dest.ref` == `self.id`.
    2. Verify `dest.position` == `0`. This is position of value in this output. TBD: this is double-checked by the input.
8. If tx version is known, all `ext_hash`es must be hashes of empty strings.
9. Insert `self.id` in utxo set.

## Retirement

    entry {
        type="retirement1"
        content:
            source:         ValueSource
            reference:      Pointer<Data>
            ext_hash:       Hash
        witness:
            ext_hash:       Hash
    }

**Rules:**

1. Let `preventry` be the `source.ref`.
2. `preventry` must be present and valid.
3. Validate `source.amount` is in range.
4. Need to ensure that this output exclusively consumes the `preventry`'s destination.
5. If `preventry` is Input or Issuance:
    1. Let `dest` be `preventry.destination`.
6. If `preventry` is Mux:
    1. Let `dest` be `preventry.destinations[source.position]`. Fail if not found.
7. Verify previous entry's destination `dest`:
    1. Verify `dest.ref` == `self.id`.
    2. Verify `dest.position` == `0`. This is position of value in this output. TBD: this is double-checked by the input.
8. If tx version is known, all `ext_hash`es must be hashes of empty strings.



## Input

    entry {
        type="input1"
        content:
          spent_output: Pointer<Output>
          reference:    Pointer<Data>
          ext_hash:     Hash
        witness:
          destination:  ValueDestination
          arguments:    String
          ext_hash:     Hash        
    }

**Rules:**

1. Validate that `spent_output` is present in tx.
2. Validate that `spent_output` is present in UTXO set.
3. Validate that `destination.ref` has been validated (visited). TBD: !!! figure out what that actually means.
4. If `destination.ref` is a Output or Retirement `nextentry`:
    1. Validate that `destination.position` == `0` (this is value's position in the output). TBD: maybe move this to Output.
    2. Let `src` be `nextentry.source`.
5. If `destination.ref` is a Mux `nextentry`:
    1. Let `src` be `nextentry.sources[destination.position]`. Fail if not present.
6. Validate next entry’s source `src`:
    1. Validate that `src.ref` == `self.id`.
    2. Validate that `src.position` == `0` (this is value's position in the input). 
    3. Validate that `src.value` == `self.spent_output.source.value`.
7. The `spent_output.program` must evaluate to `true` with given `arguments`.
8. Remove `spent_output` from UTXO set.

NB: `spent_output` is not validated, as it was already validated in the transaction that added it to the UTXO.


## Issuance

    entry {
        type="issuance1"
        content:
          anchor:           Pointer<Anchor|Input>
          value:            AssetAmount
          reference:        Pointer<Data>
          ext_hash:         Hash
        witness:
          destination:      ValueDestination
          initial_block_id: Hash
          asset_definition: Pointer<Data>
          issuance_program: Program
          arguments:        String
          ext_hash:         Hash
    }

**Rules:**

1. Check that `value.asset_id == HASH(initial_block_id, asset_definition, issuance_program)`.
2. The `issuance_program` must evaluate to `true` with given `arguments`.
3. Validate that `destination.ref` has been validated (visited). TBD: !!! figure out what that actually means.
4. If `destination.ref` is a Output or Retirement `nextentry`:
    1. Validate that `destination.position` == `0` (this is value's position in the output). TBD: maybe move this to Output.
    2. Let `src` be `nextentry.source`.
5. If `destination.ref` is a Mux `nextentry`:
    1. Let `src` be `nextentry.sources[destination.position]`. Fail if not present.
6. Validate next entry’s source `src`:
    1. Validate that `src.ref` == `self.id`.
    2. Validate that `src.position` == `0` (this is value's position in the input). 
    3. Validate that `src.value` == `self.value`.



## Mux

    entry {
        type="mux1"
        content:        
            sources:      List<ValueSource>
            ext_hash:     Hash
        witness:          
            destinations: List<ValueDestination>
            ext_hash:     Hash
    }


**Rules:**

1. Each `source` must be unique in this entry.
2. Each item in `destinations` must be unique in this entry.
3. For each item `dest` in `destinations` at position `i`:  
    1. Validate that `dest.ref` has been validated (visited). TBD !!!!
    2. If `dest.ref` is a Output or Retirement `nextentry`:
        1. Validate that `dest.position` == `0` (this is value's position in the output). TBD: maybe move this to Output.
        2. Let `src` be `nextentry.source`.
    3. If `dest.ref` is a Mux `nextentry`:
        1. Let `src` be `nextentry.sources[dest.position]`. Fail if not present.
    4. Validate next entry’s source `src`:    
        1. Validate that `src.ref` == `self.id`.
        2. Validate that `src.position` == `i` (this is value's position in this mux's destinations list). 
        3. Validate that `src.value` == `dest.value`.
        4. Pull the AssetAmount `a` from that `src` and assign it to the `dest`.
4. For each asset ID in the `sources` and `destinations`:
    1. Sum the amounts of that asset ID in `sources` and the amounts of that asset ID in `destinations`.
    2. Test that both the `sources` and `destinations` sums are less than 2^63.
    3. Test that the `sources` sum equals the `destinations` sum.
    4. Check that there is at least one `source` with that asset ID.




## ValueSource

    struct {
        ref:      Pointer<Issuance|Input|Mux>
        value:    AssetAmount
        position: Integer
    }

**Rules:**

1. TBD

## ValueDestination

    struct {
        ref:       Pointer<Output|Retirement|Mux>
        position:  Integer
    }

**Rules:**

1. TBD


## Anchor

    entry {
        type="anchor1"
        content:
          program:    Program
          timerange:  Pointer<TimeRange>
          ext_hash:   Hash
        witness:
          arguments:  String
          ext_hash:   Hash
    }

**Rules:**

1. If tx version is known, the ext fields must be empty.
2. The ID of the anchor must be globally unique on the blockchain.
3. The `program` must evaluate to `true` with given `arguments`.


## TimeRange

    entry {
        type="timerange"
        content:
            mintime:  Integer
            maxtime:  Integer
            ext_hash: Hash
        witness:
            ext_hash: Hash
    }

**Rules:**

1. `mintime` must be equal to or less than the `header.mintime` specified in the transaction header.
2. `maxtime` must be equal to or greater than the `header.maxtime` specified in the transaction header.


## AssetAmount

This is not a separate entry, but an inlined struct.

    Struct {
        assetID: Hash
        amount:  Integer
    }

## Program

This is not a separate entry, but an inlined struct that carries VM version and the program code in VM-specific encoding.

    Struct {
        vm_version: Integer
        code:       String
    }

## Pointer

Pointer is:

1. encoded as `Hash`,
2. identifies another entry by its ID,
3. restricts the possible acceptable types.

Pointer can be `nil`, in which case it is represented by all-zero 32-byte hash `0x000000...`.

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

    Pointer = Hash (identifies another entry)
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

1. Let `oldtx` be the transaction in old format.
2. Let `newtx` be a new instance of `Header` entry.
3. Let `container` be the container for all entries.
4. Set `newtx.version` to `oldtx.version`.
5. If `oldtx.reference_data` is non-empty:
    1. Let `refdata` be a new `Data` entry.
    2. Set `refdata.content` to `tx.reference_data`.
    3. Add `refdata.id` to `newtx.references`.
    4. Add `refdata` to the `container`.
6. Set `newtx.mintime` to `oldtx.mintime`.
7. Set `newtx.maxtime` to `oldtx.maxtime`.
8. Let `mux` be a new `Mux` entry.
9. For each issuance input `oldis`:
    1. Let `is` be a new `Issuance` entry.
    2. Set `is.assetid` to `oldis.assetid`.
    3. Set `is.amount` to `oldis.amount`.
    4. If `nonce` is empty:
        1. Set `is.anchor` to the first spend input of the `oldtx`.
    5. If `nonce` is non-empty:
        1. Let `a` be a new `Anchor` entry.
        2. Set `a.program` to `VM1, PUSHDATA(nonce)`.
        3. Let `tr` be a new `TimeRange` entry.
        4. Set `tr.mintime` to `oldtx.mintime`.
        5. Set `tr.maxtime` to `oldtx.maxtime`.
        6. Set `a.timerange` to `tr.id`.
        7. Set `is.anchor` to `a.id`.
        8. Add `a` to `container`.
        9. Add `tr` to `container`.
    6. Set `is.initial_block_id` to `oldis.initial_block_id`.
    7. Set `is.issuance_program` to `oldis.issuance_program` (with its VM version).
    8. Set `is.arguments` to `oldis.arguments`.
    9. If `oldis.asset_definition` is non-empty:
        1. Let `adef` be a new `Data` entry.
        2. Set `adef.content` to `oldis.asset_definition`.
        3. Set `is.asset_definition` to `adef.id`.
        4. Add `adef` to `container`.
    10. If `oldis.asset_definition` is empty:
        1. Set `is.asset_definition` to a nil pointer `0x000000...`.
    11. Add `is.id` to `mux.sources`.
    12. Add `is` to `container`.
10. For each spend input `oldspend`:
    1. Let `inp` be a new `Input` entry.
    2. Set `inp.spent_output` to `oldspend.output_id`.
    3. Set `inp.reference_data` to a nil pointer `0x00000...`.
    4. Set `inp.arguments` to `oldspend.arguments`.
    5. Add `inp.id` to `mux.sources`.
    6. Add `inp` to `container`.
11. For each output `oldout`:
    1. If the `oldout` contains a retirement program:
        1. Let `dest` be a new `Retirement` entry.
    2. If the `oldout` is not a retirement:
        1. Let `dest` be a new `Output` entry.
        2. Set `dest.control_program` to `oldout.control_program` (with its VM version).
    3. Set `dest.source` to `mux.id`.
    4. Set `dest.position` to current number of `mux.destinations` (incremented once we add it there).
    5. Add `dest.id` to `mux.destinations`.
    6. Set `dest.asset_id` to `oldout.asset_id`.
    7. Set `dest.amount` to `oldout.amount`.
    8. If `oldout.reference_data` is non-empty:
        1. Let `data` be a new `Data` entry.
        2. Set `data.content` to `oldout.reference_data`.
        3. Set `dest.reference_data` to `data.id`.
        4. Add `data` to `container`.
    9. Add `dest` to `container`.
12. For each input or issuance in `mux.sources`:
    1. Set `source[i].destination` to `mux.id`.
13. For each input or issuance in `mux.sources`:
    1. Set `source[i].destination` to `mux.id`.




### 2. NewTx -> OldTx

This is a second intermediate step that allows keeping old SDK, but refactoring how txs are represented and stored internally in Core.

TODO: ...


## Compression

1. Serialization prefix indicates the format version that specifies how things are serialized and compressed.
2. Replace hashes with varint offsets, reconstruct hashes in real time and then verify that top hash matches the source (e.g. merkle tree item)
3. Replace some repeated elements such as initial block id with indices too.
