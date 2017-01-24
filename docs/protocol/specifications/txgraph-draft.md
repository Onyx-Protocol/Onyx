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
    Predicate
    TimeConstraint
    AbstractEntry = all other types

Abstract Entry:

    - type
    - content
    - witness

    ID(entry)  = Hash("txnode" || entry.type || entry.content)
    WID(entry) = TODO: specify how to collect all WIDs from content references - every node must specify how to walk prev witnesses

Header:

    - type=header
    - content:
        version
        results: List<Output,
                      Retirement,
                      AbstractEntry>
        reference: List<ReferenceData>
        ext
    - witness:
        mintime
        maxtime
        ext

    Rules:
    1. If version is known, all exts must empty, AbstractEntries are not allowed in pointers.
    2. Results must contain at least one item.
    3. Every result must be present and valid.
    4. mintime must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
    5. maxtime must be either zero or a timestamp lower than the timestamp of the block that includes the transaction.

ReferenceData:

    - type=refdata1
    - content: blob
    - witness: ext
    
    Rules:
    1. ext must be empty

Output:

    - type=output1
    - content:
        source: Mux
        position: Integer
        reference_data: ReferenceData
        asset_id
        amount
        control_program: Program
        ext
    - witness:
        ext
        
    Rules:
    1. `amount` is in range.
    2. `source` must be present and valid.
    3. `source.destinations[position]` must equal self.id.
    4. if tx version is known, all ext fields must be empty.
    5. Insert `self.id` in utxo set.

Retirement:

    - type=retirement1
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

    - type=input1
    - content:
        spent_output: Output
        ext
    - witness:
        predicate: Predicate
        destination: AbstractEntry
        ext
        
    Rules:
    1. Verify that `predicate.program` equals `spent_output.control_program`.
    2. Validate that `predicate` is present in tx and valid.
    3. `spent_output` must be present in tx and UTXO set 
        NB: it is not validated, as it was already validated in the transaction that added it to the UTXO.
    4. Remove `spent_output` from UTXO set.

Issuance:

    - type=issuance1
    - content:
        anchor: Anchor|Input
        asset_id
        amount
        ext
    - witness:
        initial_block_id
        asset_definition
        issuance_predicate: Predicate
        destination: Any
        ext
        
    Rules:
    1. Check that asset_id == Hash(initial_block_id, asset_definition, issuance_predicate.program).
    2. `issuance_program` must be present in tx and valid, and its `caller` must be a reference to this entry.

Anchor:

    - type=anchor1
    - content:
        predicate: Predicate
        timeconstraint: TimeConstraint
        ext
    - witness:
        ext

    Rules:
    1. If tx version is known, the ext fields must be empty.
    2. The ID of the anchor must be globally unique on the blockchain.
    3. The predicate must be valid and included in the tx, and its `caller` must be a reference to this entry.

Predicate:

    - type=predicate
    - content:
        program: Program
    - witness:
        caller: AbstractEntry
        arguments
        ext
        
    Rules:
    1. If the tx version is known, the program.vm_version must be known.
    2. If program.vm_version is known, instantiate VM with that version, evaluate `program.code` with given arguments. 
       VM must return `true`.

Mux:

    - type=mux1
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
        1. Sum the source amounts of that asset (`sources[i].spent_output.amount`) and sum the destination amounts of that asset.
        2. Test that both source and destination sums are less than 2^63.
        3. Test that the source sum equals the destination sum.
        4. Check that there is at least one source with that asset ID.


TimeConstraint:

    - type=timeconstraint
    - content:
        mintime: integer
        maxtime: integer
        ext
    - witness:
        ext
    
    Rules:
    1. mintime must be equal to or greater than the mintime specified in the transaction header.
    2. maxtime must be equal to or less than the maxtime specified in the transaction header.

Program: not a separate entry, but an inlined struct
    
    - vm_version: int
    - code:       string


## Translation Layer

### 1. OldTx -> NewTx

This is a first intermediate step that allows keeping old SDK, old tx index and data structures within Core, but refactoring how txs and outputs are hashed for UTXO set and merkle root in block headers.

TODO: ...

### 2. NewTx -> OldTx

This is a second intermediate step that allows keeping old SDK, but refactoring how txs are represented and stored internally in Core.

TODO: ...
