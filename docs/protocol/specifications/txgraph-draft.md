# Transaction graph structure


Entry types:
    
    Header
    Issuance
    Input
    Output
    Retirement
    Mux
    ReferenceData
    Anchor
    AbstractEntry = all other types

Abstract Entry:

    - type
    - content
    - witness
        
    ID(entry)  = Hash(entry.type || entry.content)
    WID(entry) = Hash(ID(entry)  || entry.witness)

Header:
    
    - type=0
    - content:
        version
        results: List<Output,
                      Retirement,
                      AbstractEntry>
        reference: List<ReferenceData>
        mintime
        maxtime
        ext
    - witness:
        ext
    
    Rules:
    1. If version is known, all exts must empty, AbstractEntries are not allowed in pointers.
    2. Results must contain at least one item.
    3. Every result must be present and valid.

ReferenceData:
    
    - type=1
    - content: blob
    - witness: ext
    
    Rules:
    1. ext must be empty

Output:
    
    - type=2
    - content:
        source: Mux
        position: Integer
        reference_data: ReferenceData
        asset_id
        amount
        vm_version
        control_program
        ext
    - witness:
        ext
        
    Rules:
    1. `amount` is in range.
    2. `source` must be present and valid.
    3. `source.destinations[position]` must equal self.id.
    4. if tx version is known, all ext fields must be empty.

Retirement:

    - type=3
    - content:
        source: Mux
        position: Integer
        reference_data: ReferenceData
        asset_id
        amount
        ext
    - witness:
        ext
        
    Rules:
    1. `amount` is in range
    2. `source` must be present and valid
    3. `source.destinations[position]` must equal self.id.
    4. if tx version is known, all ext fields must be empty.

Input:
    
    - type=4
    - content:
        spent_output: Output
        ext
    - witness:
        arguments
        destination: AbstractEntry
        wext
    
    Rules:
    1. If tx version is known, disallow unknown `spent_output.vm_version`.
    2. If `spent_output.vm_version` is known, verify `spent_output.control_program` with `arguments`.
    3. `spent_output` must be present in tx and UTXO set (NB: it is not validated as it's already validated).

Issuance:
    
    - type=5
    - content:
        anchor: Anchor|Input
        asset_id
        amount
        ext
    - witness:
        initial_block_id
        asset_definition
        vm_version
        issuance_program
        arguments
        destination: AbstractEntry
        ext
    
    Rules:
    1. Check that asset_id == Hash(initial_block_id, assetdef, vm_version, issuance_program).
    2. If tx version is known, `vm_version` must be known and ext fields must be empty.
    3. If `vm_version` is known, verify `issuance_program` with `arguments`.

Anchor:
    
    - type=6
    - content:
        nonce
        ext
    - witness:
        ext
    
    Rules: 
    1. If tx version is known, the ext fields must be empty.
    2. Hash(Nonce||tx.mintime||tx.maxtime) must be globally unique.


Mux:

    - type=7
    - content:
        sources: List<Issuance|Input>
        ext
    - witness:
        destinations: List<Output|Retirement>
        ext
        
    Rules:
    1. For each source: `sources[i].destination` must equal self.id - prevents double-spending.
    2. Each source must be unique in the `sources` list, no repetitions allowed.
    3. Each identifier in `destinations` must be unique (no repetitions) and included in the transaction.
    4. For each asset on the sources and destinations:
        1. Sum the source amounts of that asset and sum the destination amounts of that asset.
        2. Test that both source and destination sums are less than 2^63.
        3. Test that the source sum equals the destination sum.
        4. Check that there is at least one source with that asset ID.

TODO: how to update UTXO:
1. throw out visited input.output_ids from UTXO set.
2. insert visited output.ids (mentioned in tx) to UTXO set.


## Translation Layer

### 1. OldTx -> NewTx

This is a first intermediate step that allows keeping old SDK and old tx indexer, but refactoring how txs and outputs are hashed.



### 2. NewTx -> OldTx

This is a second intermediate step that allows keeping old SDK, but refactoring how txs are stored internally in Core.










