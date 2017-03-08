## Translation Layer

(This is a temporary guide for translating between old-style transaction data structures and new-style transaction data structures.)

### OldTx -> NewTx

This is a first intermediate step that allows keeping old SDK, old tx index and data structures within Core, but refactoring how txs and outputs are hashed for UTXO set and merkle root in block headers.

1. Let `oldtx` be the transaction in old format.
2. Let `newtx` be a new instance of `TxHeader` entry.
3. Let `container` be the container for all entries.
4. Set `newtx.version` to `oldtx.version`.
5. If `oldtx.data` is non-empty:
    1. Let `refdata` be a new `Data` entry.
    2. Set `refdata.body` to `tx.data`.
    3. Set `newtx.data` to `refdata.id`.
    4. Add `refdata` to the `container`.
6. Set `newtx.mintime` to `oldtx.mintime`.
7. Set `newtx.maxtime` to `oldtx.maxtime`.
8. Let `mux` be a new `Mux` entry.
9. For each old input `oldinp`:
    1. If the old input is issuance:
        1. Let `is` be a new `Issuance` entry.
        2. Set `is.value` to `AssetAmount { oldinp.assetid, oldinp.amount }`.
        3. If `nonce` is empty:
            1. Set `is.anchor` to the ID of the first new input. (If no input was mapped yet, go back to this step when such input is added.)
        4. If `nonce` is non-empty:
            1. Let `a` be a new `Nonce` entry.
            2. Set `a.program` to (VM1, `PUSHDATA(nonce) DROP ASSET PUSHDATA(oldinp.assetid) EQUAL`. (The program pushes the nonce onto the stack then drops it, then calls the ASSET opcode, pushes the hardcoded old asset ID onto the stack, and checks that they are equal.)
            3. Let `tr` be a new `TimeRange` entry.
            4. Set `tr.mintime` to `oldtx.mintime`.
            5. Set `tr.maxtime` to `oldtx.maxtime`.
            6. Set `a.timerange` to `tr.id`.
            7. Set `is.anchor` to `a.id`.
            8. Add `a` to `container`.
            9. Add `tr` to `container`.
        5. Set `is.initial_block_id` to `oldinp.initial_block_id`.
        6. Set `is.issuance_program` to `oldinp.issuance_program` (with its VM version).
        7. If `oldinp.asset_definition` is non-empty:
            1. Let `adef` be a new `Data` entry.
            2. Set `adef.body` to `oldinp.asset_definition`.
            3. Set `is.asset_definition` to `adef.id`.
            4. Add `adef` to `container`.
        8. If `oldinp.asset_definition` is empty:
            1. Set `is.asset_definition` to a nil pointer `0x000000...`.
        9. Create `ValueSource` struct `src`:
            1. Set `src.ref` to `is.id`.
            2. Set `src.position` to 0.
            3. Set `src.value` to `is.value`.
            4. Add `src` to `mux.sources`.
        10. Add `is` to `container`.
    2. If the old input is a spend:
        1. Let `inp` be a new `Spend` entry.
        2. Set `inp.spent_output` to `oldinp.output_id`.
        3. Set `inp.data` to a nil pointer `0x00000...`.
        4. Create `ValueSource` struct `src`:
            1. Set `src.ref` to `inp.id`.
            2. Set `src.position` to 0.
            3. Set `src.value` to `AssetAmount{ oldinp.spent_output.(assetid,amount) } `.
            4. Add `src` to `mux.sources`.
        5. Add `inp` to `container`.
11. For each output `oldout` at index `i`:
    1. If the `oldout` contains a retirement program:
        1. Let `destentry` be a new `Retirement` entry.
    2. If the `oldout` is not a retirement:
        1. Let `destentry` be a new `Output` entry.
        2. Set `destentry.control_program` to `oldout.control_program` (with its VM version).
    3. Create `ValueSource` struct `src`:
        1. Set `src.ref` to `mux.id`.
        2. Set `src.position` to `i`.
        3. Set `src.value` to `AssetAmount { oldout.asset_id, oldout.amount }`.
        4. Set `destentry.source` to `src`.
    4. If `oldout.data` is non-empty:
        1. Let `data` be a new `Data` entry.
        2. Set `data.body` to `oldout.data`.
        3. Set `destentry.data` to `data.id`.
        4. Add `data` to `container`.
    5. Add `destentry` to `container`.
    6. Add `destentry` to `newtx.results`.


### OldTxID -> NewTxID

1. Map old tx to `newtx`.
2. Return new tx's header ID as NewTxID.

### OldWitTxID -> NewWitTxID

1. Map old tx to new tx.
2. Return new tx's header ID as NewWitTxID. This is the same as NewTxID.

### OldOutputID -> NewOutputID

When indexing old tx's outputs:

1. Map old tx to new tx.
2. Take corresponding new output.
3. Compute its entry ID which will be NewOutputID.
4. Use this new output ID to identify unspent outputs in the DB.

### OldUnspentID -> NewUnspentID

When inserting old tx's outputs into UTXO merkle set:

1. Map old tx to new tx.
2. Take corresponding new output.
3. Compute its entry ID which will be NewUnspentID. (This is the same as NewOutputID.)
4. Use this new unspent ID to insert into UTXO merkle set.


### OldIssuanceHash -> NewIssuanceHash

1. Map old tx to new tx.
2. For each nonce entry in the new tx:
    1. check its time range is within network-defined limits (not unbounded).
    2. Use this entry ID as NewIssuanceHash
    3. Insert new issuance hash in the current _issuance memory_ annotated with expiration date based on `nonce.timerange.maxtime`.

### OldSigHash -> NewSigHash

1. Map old tx to new tx.
2. For each entry where a program is evaluated (Spend, Issuance or Nonce):
    1. Compute `sighash = HASH(txid || entryid)`.



### NewTx -> OldTx

This is a second intermediate step that allows keeping old SDK, but refactoring how txs are represented and stored internally in Core.

TODO: ...


## Compression

1. Serialization prefix indicates the format version that specifies how things are serialized and compressed.
2. Replace hashes with varint offsets, reconstruct hashes in real time and then verify that top hash matches the source (e.g. merkle tree item)
3. Replace some repeated elements such as initial block id with indices too.

### VM mapping

This shows how the implementation of each of the VM instructions need to be changed. Ones that say "no change" will work as already implemented on the OLD data structure. 

* CHECKOUTPUT:   no change
* ASSET:         no change
* AMOUNT:        no change
* PROGRAM:       no change
* MINTIME:       no change
* MAXTIME:       no change
* INDEX:         no change
* NONCE:         eliminated
* TXREFDATAHASH: `newtx.refdatahash()`
* REFDATAHASH:   `newcurrentinput.refdatahash()`
* TXSIGHASH:     `hash(newcurrentinput.id() || newtx.id())`
* OUTPUTID:      `newcurrentinput.spent_output.id()`


New opcodes:

* ENTRYID:       `currententry.id()`
* NONCE:         `currentissuance.anchor.id()` (fails if the entry is not an issuance)


### Block header format

The new slightly different serialization format (i.e. the type prefix and extension hash format) should be applied to the block header as well. We are also removing the block witness from the block ID, as discussed above. Finally, we should flatten the confusing "block commitment" and simply make it three separate fields in the block header.

#### BlockHeader entry

    entry {
        type="blockheader"
        body:
            version:                Integer
            height:                 Integer
            previous_block_id:      Pointer<BlockHeader>
            timestamp:              Integer
            transactions:           MerkleTree<Pointer<TxHeader>>
            assets:                 PatriciaTree<Pointer<Output>>
            next_consensus_program: String
            ext_hash:               Hash
        witness:
            ext_hash:               Hash        
    }

The `MerkleTree` and `PatriciaTree` types are just 32-byte hashes representing the root of those respective trees.

#### OldBlockHeader -> NewBlockHeader

This generates a new BlockHeader data structure, for hashing purposes, from an old block.

1. Let `oldblock` be the block in old format.
2. Let `newblock` be a new instance of `BlockHeader` entry.
3. Set `newblock.version` to `oldblock.version`.
4. Set `newblock.height` to `oldblock.height`.
5. Set `newblock.previous_block_id` to `oldblock.previous_block_id`.
6. Set `newblock.timestamp` to `oldblock.timestamp`.
7. Set `newblock.transactions` to `oldblock.block_commitment.transactions` (i.e. the root of the Merkle tree). Note that this Merkle tree should have been calculated using the new transaction ID.
7. Set `newblock.assets` to `oldblock.block_commitment.assets` (i.e. the root of the Patricia tree). Note that this Patricia tree should have been calculated using the new Output IDs.
8. Set `newblock.next_consensus_program` to `oldblock.block_commitment.next_consensus_program`

#### VM mapping

PROGRAM:       same
NEXTPROGRAM:   same
BLOCKTIME:     same
BLOCKSIGHASH:  newblock.id()
