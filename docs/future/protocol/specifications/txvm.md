# TxVM

This is the specification for TxVM, which combines a representation for blockchain transactions with the rules for ensuring their validity.

* [Motivation](#motivation)
* [TxVM operation](#txvm-operation)
* [Compatibility](#compatibility)
* [Types](#types)
* [Encoding](#encoding)
* [Stacks](#stacks)
* [Instructions](#instructions)
* [Examples](#examples)

## Motivation

Earlier versions of Chain Core represented transactions with a static data structure, exposing the pieces of information needed to test the transaction’s validity. A separate set of validation rules could be applied to that information to get a true/false result.

Under TxVM, these functions are combined in such a way that an executable program string is both the transaction’s representation and the proof of its validity.

When the virtual machine executes a TxVM program, it accumulates different types of data on different stacks. This data corresponds to the information exposed in earlier versions of the transaction data structure: inputs, outputs, time constraints, nonces, and so on. Under TxVM, that information is _only_ available as a result of executing the program, and the program only completes without error if the transaction is well-formed (i.e., its inputs and outputs balance, prevout control programs are correctly satisfied, etc). No separate validation steps are required.

The pieces of transaction information - the inputs, outputs, etc. - that are produced during TxVM execution are also _consumed_ in order to produce the transaction summary, which is the sole output of a successful TxVM program. To capture pieces of transaction information for purposes other than validation, TxVM implementations can and should provide callback hooks for inspecting and copying data from the various stacks at key points during execution.



## TODO

---------------------------------------------------------------------------------

### Review extensibility features (adding new fields to predefined types)

1. See [Extend](#extend) opcode.

### Tx merkle root - should be txid or the entire tx.

1. We need to commit to tx script and runlimit to prevent DoS attacks where clients execute modified programs and waste CPUs.
2. We need to commit to txid, so clients can inspect effects of the transaction w/o even transmitting full script and bulky signatures/rangeproofs.

Ideal merkle item contains the following data (this is a bit redundant and presented for illustration):

    merkle item = hash(txid || runlimit || hash(txscript) || hash(effects))

* `txid` is a canonical ID for use in the external systems. It should itself include `version,runlimit,effects` and must exclude `txscript` to not be malleable.
* `runlimit` is necessary to prevent large runlimit attacks (block signers sign over the committed runlimit)
* `txscript` is a witness of how transaction is formed and validated - contains all bulky signatures and rangeproofs
* `effects` is an actual outcome of the transaction that's needed for state updates (necessary for blockchain state machine, even if txscript is stripped)

This allows fetching raw txscript w/o redundant raw effects, or raw effects w/o bulky txscript (aka witness).

#### Suggestion 1: 

* TxSummary = {version, runlimit, hash(effects)}
* TxID = hash(TxSummary)
* TxWitness = {TxID, hash(Tx)}
* Tx = {version, runlimit, txscript}

The "tx" entity will be raw transaction, which can be transformed into TxSummary and to TxWitness. TxWitness is committed to the merkle root, from where one can extract
either TxSummary (via TxID) or raw transaction via its hash.

#### Suggestion 2:

Same as above structurally, but changing the names so TxID is not confused with hash(Tx) (as in suggestion above).

    s/TxSummary/Tx/ => result of the txvm is a transaction object; allows saying `hash(TxSummary) == TxID`
    s/Tx/TxWitness/ => raw script+version+runlimit is the witness, that yields a transaction
    merkle item = SHA3-256(Tx.ID || TxWitness.ID) - simple: commit witness & transaction, symmetrical: can fetch either one of them.




### Legacy scripts and legacy asset IDs

1. Need to allow unlocking legacy outputs.
2. Need to issue legacy asset IDs in the new txvm transactions.
3. Need to create issuance candidates from the legacy asset IDs.

### Isolated program execution

Use case: smart contract needs to defer authorization to an opaque program and guarantee that it does not mess with the intermediate state in unauthorized manner.

Ideas:

1. Provide data and values to the program (sandboxed execution) - most robust
2. Have isolated altstack per command, move sensitive items over there before executing other commands - bandaiding

Problems:

1. How do we ensure the program is flexible enough? The nested program might also want to consume some inputs, issue something etc.

---------------------------------------------------------------------------------








## TxVM operation

Validation of the transaction happens in a context of a validating a block of transactions. Large part of that validation is handled by the TxVM logic with a few validation rules outside of it.

### VM state

TxVM is a state machine consisting of:

1. [Stacks](#stacks):
  0. Data stack
  1. Alt stack
  2. Entry stack
  3. Command stack
  4. Effect stack
2. Extension flag (boolean)
3. Runlimit (int64).
4. [Transaction](#transaction) tuple.

### VM Execution

TODO: make Transaction tuple or at least its version introspectable by programs. TxSummary must include version

1. The VM is initialized with:
  * all [stacks](#stacks) empty,
  * `extension` flag set to true or false according to [transaction versioning](#versioning) rules,
  * `runlimit` set to the runlimit specified by the [transaction tuple](#transaction),
  * [transaction tuple](#transaction) set to the transaction being validation.
2. TxVM bytecode is being executed according to behaviour described per each [instruction](#instructions).
3. Each instruction consumes [runlimit](#runlimit). If TxVM runs out of runlimit before the end of the execution, execution fails.
4. When the program counter is equal to the length of the program, execution is complete.
5. The top item of the [Effect stack](#effect-stack) must be a [Transaction Summary](#transaction-summary).
6. There must be no other Transaction Summaries in the Effect stack, otherwise execution fails.
7. There must be at least one [anchor](#anchor) in the Effect stack.
8. The Entry stack must be empty.

Note: remaining runlimit in TxVM could be greater than 0: excess value is allowed for future extensions.

### Post-execution

If execution and all the required checks do not fail, Effect stack is introspected and blockchain state is updated.

1. If any [Mintime](#mintime) item on the Effect stack has `mintime` greater than the block’s timestamp, reject transaction.
2. If any [Maxtime](#maxtime) item on the Effect stack has `maxtime` less than the block’s timestamp, reject transaction.
3. [Transaction ID](#transaction-id) is computed as ID committed to the block as ID of the Transaction Summary.
4. For each [Input](#input), its `contractid` is removed from the UTXO set.
5. For each [Output](#output), its `contractid` is added to the UTXO set.
6. Remove all outdated nonces from Nonce set (based on block's timestamp).
7. For each [Nonce](#nonce):
  1. Verify that `nonce.blockchainid` is equal to the current blockchain ID.
  2. Add ID of the nonce to the Nonce set.
8. TBD: update record set


### Versioning

1. Every instance of Chain Core software defines **current block version** and **current transaction version**.
2. All TxVM [Transaction](#transaction) tuples must have transaction version 2 or greater. This is to avoid confusion with version 1 transactions in the legacy format.
3. All TxVM [Block](#block) tuples must have version 2 or greater. This is to avoid confusion with version 1 blocks in the legacy format.
4. Blocks that include TxVM transactions must have version 2 or greater.
5. Each block must have the same version or greater as the previous block.

Extensions:

1. If the block version is equal to _current block version_, transaction cannot have version higher than the _current transaction version_.
2. If the transaction version is higher than the _current transaction version_, TxVM `extension` flag is set to `true`.
3. Otherwise, `extension` flag is set to `false`.


### Runlimit

Blocks commit to the total runlimit that be greater or equal to the sum of runlimits specified in all transactions within a block. Excess runlimit is allowed for future extensions.

The TxVM is initialized with a runlimit specified in [Transaction](#transaction) tuple. Each instruction reduces that number.

If the runlimit goes below zero while the program counter is less than the length of the program, execution fails.

1. Each instruction costs `1`.
2. Each instruction that pushes an item to the data stack, including as the result of an operation (such as `add`, `cat`, `merge`, `field`, and `untuple`), costs an amount based on the type and size of that data:
  1. Each string that is pushed to the stack costs `1 + len`, where `len` is the length of that string in bytes.
  2. Each number that is pushed to the stack costs `1`.
  3. Each tuple that is pushed to the stack costs `1 + len`, where `len` is the length of that tuple.
3. Each instruction that pushes an item to any stack other than the data or alt stack costs `256` for each item so pushed.
4. Each `checksig` and `pointmul` instruction costs `1024`. [TBD: estimate the actual cost of these instruction relative to the other instructions].
5. Each `roll`, `bury`, or `reverse` instruction costs `n`, where `n` is the `n` argument to that operation.

Execution of the transaction can leave some runlimit unconsumed: excess runlimit is allowed for future extensions.

## Compatibility

TxVM transactions are not compatible with version 1 transactions. However, they allow interacting with pre-existing blockchain state: nonces, outputs and asset IDs.

### Spending legacy outputs

See [UnlockLegacy](#unlocklegacy) instruction that allows unlocking value stored in [legacy outputs](#legacy-output).

### Issuance of legacy Asset ID

See [IssueLegacy](#issuelegacy) instruction that allows issuing value based on legacy asset IDs.

### Confidential issuance of legacy Asset IDs

See [LegacyIssuanceCandidate](#legacyissuancecandidate) instruction that allows creating an issuance candidate for the legacy asset IDs.

### Soft-fork and hard-fork upgrades to TxVM

Blocks, transactions and TxVM instructions are designed with extensibility in mind.

Upgrades can be done via hard forks and soft forks.

TBD: Need to specify how soft/hard fork upgrades are possible with NOPs and Extend opcode.



## Types

There are three types of items on the VM stacks, with the following numeric identifiers.

* Int64 (33)
* String (34)
* Tuple (35)

"Boolean" is not a separate type. Operations that produce booleans produce the two int64 values `0` (for false) and `1` (for true). Operations that consume booleans treat `0` as false, and all other values (including all strings and tuples) as `true`.

### Int64

An integer between 2^-63 and 2^63 - 1.

### String

A bytestring with length between 0 and 2^31 - 1 bytes.

### Item IDs

The ID of an item is the SHA3 hash of `"txvm" || encode(item)`, where `encode` is the [encode](#encode) operation.

### Transaction ID

The ID of a [Transaction Summary](#transaction-summary) item:

    SHA3-256("txvm" || encode(summary))

### Tuple

An immutable collection of items of any type.

There are several named types of tuples.

For extensibility, each tuple may contain additional fields that are not defined yet.
These fields contribute to the tuple [ID](#item-ids), but do not affect the execution of TxVM.

### Block

1. `type`, a string, "block"
2. `version`, an int64
3. `previous`, a string, 32-byte ID of the previous block (or hash of a legacy block)
4. `predicate`, a tuple of 1 or more tuples of 1 one or more [Multisig Predicates](#multisig-predicate). Outer tuple is OR function, inner tuples are AND functions of the multisig predicates.
5. `runlimit`, an int64
6. `txroot`, a string, a merkle root of a set of all transactions included in the block
7. TBD: UTXO & nonces set merkle root
8. TBD: records set merkle root (maybe the same root as utxo and nonces?)

### Signed Block

1. `type`, a string, "signedblock"
2. `block`, a tuple of type [Block](#block)
3. TBD: `signatures`, a tuple of tuples of tuples of signatures matching the format of the predicate.

Note: Signed Block is used to encode signatures for the block. The ID of the Signed Block is not used to prevent malleability of the blocks.

### Multisig Predicate

1. `threshold`, an int64
2. `pubkeys`, a tuple of [public keys](#public-key)

### Transaction

0. `type`, a string, "tx"
1. `version`, an int64
2. `runlimit`, an int64
3. `program`, a string

Note: ID of this Transaction tuple is not the same as [Transaction ID](#transaction-id) which is computed after TxVM is evaluated.

### Transaction Witness

0. `type`, a string, "txwitness"
1. `txid`, a string
2. `programhash`, a string (SHA3-256 hash of a program)

TBD: alternatively, a hash(transaction).


### Value

0. `type`, a string, "value"
1. `amount`, an int64
2. `assetID`, a string

### Value Commitment

0. `type`, a string, "valuecommitment"
1. `valuepoint`, first half of [value commitment](ca.md#value-commitment) as described in [CA](ca.md) specification.
2. `blindingpoint`, second half of [value commitment](ca.md#value-commitment) as described in [CA](ca.md) specification.

### Asset Commitment

0. `type`, a string, "assetcommitment"
1. `assetpoint`, first half of [asset ID commitment](ca.md#asset-id-commitment) as described in [CA](ca.md) specification.
2. `blindingpoint`, second half of [asset ID commitment](ca.md#asset-id-commitment) as described in [CA](ca.md) specification.

### Unproven Value

0. `type`, a string, "unprovenvalue"
1. `valuecommitment`, a [value commitment](#value-commitment)

### Proven Value

0. `type`, a string, "provenvalue"
1. `valuecommitment`, a [value commitment](#value-commitment)
2. `assetcommitment`, an [asset commitment](#asset-commitment)

### Record type

0. `type`, a string, "record"
1. `commandprogram`, a string
2. `data`, an item

### Input

0. `type`, a string, "input"
1. `contractid`, a string

### Output

0. `type`, a string, "output"
1. `contractid`, a string

### Read

0. `type`, a string, "read"
1. `contractid`, a string

### Contract

0. `type`, a string, "contract"
1. `values`, a tuple of either [values](#values) or [proven values](#proven-values)
2. `program`, a [Program](#program)
3. `anchor`, a string

### Program

0. `type`, a string, "program"
1. `program`, a string

### Nonce

0. `type`, a string, "nonce"
1. `program`, a string
2. `mintime`, an int64
3. `maxtime`, an int64
4. `blockchainid`, a string

### Anchor

0. `type`, a string, "anchor"
1. `value`, a string

### Retirement

0. `type`, a string, "retirement"
1. `value`, a [value commitment](#value-commitment)

### Asset Definition

0. `type`, a string, "assetdefinition"
1. `issuanceprogram`, a [Program](#program)

### Issuance Candidate

0. `type`, a string, "issuancecandidate"
1. `assetID`, a string
2. `issuanceKey`, a [Public Key](#public-key)

### Maxtime

0. `type`, a string, "maxtime"
1. `maxtime`, an int64

### Mintime

0. `type`, a string, "mintime"
1. `mintime`, an int64

### Annotation

0. `type`, a string, "annotation"
1. `data`, a string

### Transaction Summary

0. `type`, a string, "transactionSummary"
1. `version`, an int64
2. `runlimit`, an int64
3. `effecthash`, a 32-byte hash of all the effect entries

### Legacy Output

0. `sourceID`, a 32-byte ID
1. `assetID`, a 32-byte asset ID
2. `amount`, an int64
3. `index`, an int64
4. `program`, a string
5. `data`, a string


## Encoding

### Varint

TODO: Describe rules for encoding and decoding unsigned varints.

### Point

See [Point](ca.md#point) definition in Confidential Assets specification.

### Point Pair

See [Point Pair](ca.md#point-pair) definition in Confidential Assets specification.

### Tuple Encoding

Tuples are encoded using bytecode that produces such tuple on the data stack when executed.

See [encode](#encode) operation for details.




## Stacks

### Stack identifiers

0. Data stack
1. Alt stack
2. Entry stack
3. Command stack
4. Effect stack

### Data stack

Items on the data stack can be int64s, strings, or tuples.

### Alt stack

Items on the alt stack have the same types as items on the data stack. The alt stack starts out empty. Items can be moved from the data stack to the alt stack with the [toaltstack](#toaltstack) instruction, and from the alt stack to the data stack with the [fromaltstack](#fromaltstack).

### Entry stack

Items on the Entry stack are [Values](#value).

### Command stack

Items on the Command stack are currently executed [Programs](#program).

### Effect stack

Items on the Effect stack are [Inputs](#input), [Outputs](#output), [Reads](#read), [Nonces](#nonce), [Retirements](#retirement).






## Instructions

## Control flow operations

### Fail

Halts VM execution, returning `false`.

### PC

Pushes the current program counter (after incrementing for this instruction) to the data stack.

### JumpIf

1. Pops an integer `destination` from the data stack.
2. Pops a boolean `cond` from the data stack.
3. If `cond` is false, do nothing.
4. If `cond` is true, set program counter to `destination`.
5. Fail if `destination` is negative.
6. Fail if `destination` is greater than the length of the current program.

Note 1: normally the program using `jumpif` would be written as `<cond> <destination> jumpif` in assembly, but for brevity a slightly different syntax is used:

    <cond> jumpif:$<destination>

where `<destination>` is a name of a label somewhere in the program. The label itself is marked as `$<label>` among the instructions. Example:

    <cond> jumpif:$xyz  ... $xyz ...

Note 2: unconditional jump can be implemented with `jumpif` prefixed with "push 1" opcode:

    1 jumpif:$<destination>


## Stack operations

General stack operations intentionally do not allow adding or removing arbitrary items. Only data stack and altstack allow arbitrary manipulations (see [Data stack operations](#data-stack-operations)), other stacks employ specific validation rules for adding or removing individual elements.

### Roll

1. Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks).
2. Pops another integer `n` from the data stack.
3. Fails if `stackid` is not Data stack, Alt stack or Entry stack.
4. On the stack identified by `stackid`, moves the `n`th item from the top from its current position to the top of the stack.

Fails if `stackid` does not correspond to a valid stack, or if the stack has fewer than `n + 1` items.

### Bury

1. Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks).
2. Pops a number `n` from the data stack.
3. Fails if `stackid` is not Data stack, Alt stack or Entry stack.
4. On the stack identified by `stackid` moves the top item and inserts it at the `n`th-from-top position.

### Reverse

1. Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks).
2. Pops a number `n` from the data stack.
3. Fails if `stackid` is not Data stack, Alt stack or Entry stack.
4. On the stack identified by `stackid`:
  1. Removes the top `n` items.
  2. Inserts them back in reverse order.

### Depth

1. Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks).
2. Counts the number of items on the stack identified by `stackid`.
3. Pushes that count to the data stack.

### Peek

1. Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks).
2. Pops an integer `n` from the data stack.
3. Looks at the `n`th item of the stack identified by `stackid`, and pushes a copy of it to the data stack.




## Data stack operations

### Equal

1. Pops two items `val1` and `val2` from the data stack.
2. If they have differing types, pushes `false` to the stack.
3. If they have the same type:
  1. if they are tuples, pushes `false` to the stack;
  2. otherwise, if they are equal, pushes `true` to the stack;
  3. otherwise, pushes `false` to the stack.

### Type

1. Looks at the top item on the data stack.
2. Pushes a number to the stack corresponding to that item's [type](#type).

### Len

1. Pops a string or tuple `val` from the data stack.
2. If `val` is a tuple, pushes the number of fields in that tuple to the data stack.
3. If `val` is a string, pushes the length of that string to the data stack.
4. Fails if `val` is a number.

### Drop

Drops an item from the data stack.

### ToAlt

1. Pops an item from the data stack.
2. Pushes it to the alt stack.

### FromAlt

1. Pops an item from the alt stack.
2. Pushes it to the data stack.




## Tuple operations

### Tuple

1. Pops an integer `len` from the data stack.
2. Pops `len` items from the data stack.
3. Creates a tuple with these items on the data stack.

### Untuple

1. Pops a tuple `tuple` from the data stack.
2. Pushes each of the fields in `tuple` to the stack in reverse order (so that the 0th item in the tuple ends up on top of the stack).
3. Pushes `len`, the length of the tuple, to the data stack.

### Field

1. Pops an integer `i` from the top of the data stack.
2. Pops a tuple `tuple`.
3. Pushes the item in the `i`th field of `tuple` to the top of the data stack.
4. Fails if `i` is negative or greater than or equal to the number of fields in `tuple`.



## Boolean operations

### Not

1. Pops a boolean `p` from the stack.
2. If `p` is `true`, pushes `false`.
3. If `p` is `false`, pushes `true`.

### And

1. Pops two booleans `p` and `q` from the stack.
2. If both `p` and `q` are true, pushes `true`.
3. Otherwise, pushes `false`.

### Or

1. Pops two booleans `p` and `q` from the stack.
2. If both `p` and `q` are false, pushes `false`.
3. Otherwise, pushes `true`.



## Numeric operations

### Add

Pops two integers `a` and `b` from the stack, adds them, and pushes their sum `a + b` to the stack.
TODO: fail on overflow

### Sub

Pops two integers `a` and `b` from the stack, subtracts the second frmo the first, and pushes their difference `a - b` to the stack.
TODO: fail on overflow

### Mul

Pops two integers `a` and `b` from the stack, multiplies them, and pushes their product `a * b` to the stack.
TODO: fail on overflow

### Div

Pops two integers `a` and `b` from the stack, divides them truncated toward 0, and pushes their quotient `a / b` to the stack.
TODO: fail on overflow/divide-by-zero

### Mod

Pops two integers `a` and `b` from the stack, computes their remainder `a % b`, and pushes it to the stack.

TODO: clarify behavior. Should act like Go.

### LeftShift

Pops two integers `a` and `b` from the stack, shifts `a` to the left by `b` bits.

TODO: clarify behavior. Fail if overflows Int64.

### RightShift

Pops two integers `a` and `b` from the data stack, shifts `a` to the right by `b` bits.

TODO: clarify behavior.

### GreaterThan

Pops two numbers `a` and `b` from the data stack. If `a` is greater than `b`, pushes `true` to the stack. Otherwise, pushes `false`.



## String operations

### Cat

Pops two strings, `a`, then `b`, from the data stack, concatenates them, and pushes the result, `a || b` to the stack.

### Slice

Pops two integers, `start`, then `end`, from the data stack. Pops a string `str` from the data stack. Pushes the string `str[start:end]` (with the first character being the one at index `start`, and the last character being the one before index `end`). Fails if `end` is less than `start`, if `start` is less than 0, or if `end` is greater than the length of `str`.



## Bitwise operations

### BitNot

Pops a string `a` from the data stack, inverts its bits, and pushes the result `~a`.

### BitAnd

Pops two strings `a` and `b` from the data stack. Fails if they do not have the same length. Performs a "bitwise and" operation on `a` and `b` and pushes the result `a & b`.

### BitOr

Pops two strings `a` and `b` from the data stack. Fails if they do not have the same length. Performs a "bitwise or" operation on `a` and `b` and pushes the result `a | b`.


### BitXor

Pops two strings `a` and `b` from the data stack. Fails if they do not have the same length. Performs a "bitwise xor" operation on `a` and `b` and pushes the result `a ^ b`.




## Crypto operations

### SHA256

1. Pops a string `a` from the data stack.
2. Computes a SHA2-256 hash on it: `h = SHA2-256(a)`.
3. Pushes the resulting string `h` to the data stack.


### SHA3

1. Pops a string `a` from the data stack.
2. Computes a SHA3-256 hash on it: `h = SHA3-256(a)`.
3. Pushes the resulting string `h` to the data stack.


### CheckSig

1. Pops a string `pubKey`, a string `msg`, and a string `sig` from the data stack.
2. Performs an EdDSA (RFC8032) signature check with `pubKey` as the public key, `msg` as the message, and `sig` as the signature.
3. Pushes `true` to the data stack if the signature check succeeded, and `false` otherwise.

TODO: Should we switch order of `pubKey` and `msg`?


### PointAdd

1. Pops two strings `A` and `B` from the data stack.
2. Decodes each of them as [Ed25519 curve points](#point).
3. Performs an elliptic curve addition `C = A + B`.
4. Encodes the resulting point `C` as a string, and pushes it to the data stack.
5. Fails if `A` and `B` are not valid curve points.


### PointSub

1. Pops two strings `A` and `B` from the data stack.
2. Decodes each of them as [Ed25519 curve points](#point).
3. Performs an elliptic curve subtraction `C = A - B`.
4. Encodes the resulting point `C` as a string, and pushes it to the data stack.
5. Fails if `A` and `B` are not valid curve points.


### PointMul

1. Pops an integer `x` and a string `P` from the data stack.
2. Decodes `P` as an [Ed25519 curve point](#point).
3. Performs an elliptic curve scalar multiplication `x·P`.
4. Encodes the result as a string, and pushes it to the data stack.
5. Fails if `P` is not a valid curve point.



## Annotation operations

### Annotate

1. Pops a string, `data`, from the data stack.
2. Pushes an [Annotation](#annotation) with `data` of `data` to the Effect stack.



## Command operations

### Command

1. Pops a string `prog` from the data stack.
2. Constructs a tuple `p` of type [Program](#program) with `p.program` equal to `prog`.
3. Pushes `p` to the Command stack.
4. Executes `p.program`.
5. Pops a `p` from the Command stack.

Note 1: when step 5 is reached, all nested commands are already executed and popped, so the top item on the Command stack is the one that just finished executing.

Note 2: program can be an empty string; in such case, steps 2-5 can be omitted as they have no effect.

## Condition operations

### Defer

1. Pops a [Program](#program) from the data stack.
2. Pushes it to the Entry stack.

TODO: seems like `opcommand` should be `opdefer;opsatisfy`. We have too many entities here - programs on data stack, programs on entry stack and programs in the command stack.

### Satisfy

TBD: name "satisfy" no longer aligned with "conditions" because we now have "programs". Maybe rename to it `run`?

1. Pops a [Program](#program) from the Entry stack
2. Executes it using [command](#command) operation.



## Record operations

### Create

1. Pops an item, `data`, from the data stack.
2. Peeks at the top item on the Command stack,`command`.
3. Pushes a [Record](#record) to the Entry stack with `commandprogram` equal to `command.program` and `data` equal to `data`.


### Delete

1. Pops a Record, `record`, from the Record stack.
2. Peeks at the top item on the Command stack,`p`.
3. If `record.commandprogram` is not equal to `p.program`, fails execution.


### Complete

1. Peeks at the top item on the Record stack, `record`.
2. Peeks at the top item on the Command stack, `p`.
3. If `record.commandprogram` is not equal to `p.program`, fails execution.
4. Moves `record` to the Effect stack.



## Contract operations

### Unlock

1. Pops an item `value` of type [Value](#value) or [Proven Value](#proven-value) from the data stack.
2. Pops an item of type [Anchor](#anchor) from the data stack.
3. Peeks at the top [Program](#program) `p` on the Command stack.
4. Constructs a tuple `input` of type [Contract](#contract), with:
  * `input.program` equal to `p.program`,
  * `input.anchor` equal to `anchor.value`,
  * `input.value` equal to `value`.
5. Computes the [ID](#item-ids) `contractid` of `input`.
6. Pushes an [Input](#input) to the Effect stack with `contractid` equal to `contractid`.
7. If `value` is a [Proven Value](#proven-value), pushes `value.assetcommitment` to the Entry stack.
8. Constructs a tuple `a` of type [Anchor](#anchor) with `a.value` equal to `input.anchor`.
9. Pushes `a` to the Entry stack.
10. Pushes `value` to the Entry stack.

### Read

1. Pops an item `value` of type [Value](#value) or [Proven Value](#proven-value) from the data stack.
2. Pops an item of type [Anchor](#anchor) from the data stack.
3. Peeks at the top [Program](#program) `p` on the Command stack.
4. Constructs a tuple `contract` of type [Contract](#contract), with:
  * `contract.program` equal to `p.program`,
  * `contract.anchor` equal to `anchor.value`, and
  * `contract.value` equal to `value`.
5. Computes the [ID](#item-ids) `contractid` of `input`.
6. Pushes a [Read](#read) to the Effect stack with `contractid` equal to `contractid`.

### Lock

1. Pops an item of type [Value](#Value) or [Proven Value](#proven-value), `value`, from the Entry stack.
2. Pops an [Anchor](#anchor) `a` from the Entry stack.
3. Peeks at the top [Program](#program) `p` on the Command stack.
4. Constructs a tuple `contract` of type [Contract](#contract), with:
  * `contract.program` equal to `p.program`,
  * `contract.anchor` equal to `a`,
  * `contract.value` equal to `value`.
5. Computes the [ID](#item-id) `contractid` of `contract`.
6. Pushes an [Output](#output) to the Effect stack with `contractid` equal to `contractid`.


## Value operations

### Issue

1. Pops an int64 `amount` from the data stack.
2. Peeks at the top item on the Command stack, `p` of type [Program](#program).
3. Computes the [ID](#item-ids) `assetid` of an [asset definition](#asset-definition) tuple with `issuanceprogram` set to `p.program`.
4. Pushes a [value](#value) with amount `amount` and assetID `assetID` to Entry stack.


### Merge

1. Pops two [Values](#value) from the Entry stack.
2. If their asset IDs are different, execution fails.
3. Pushes a new [Value](#value) to the Entry stack, whose asset ID is the same as the popped values, and whose amount is the sum of the amounts of each of the popped values.


### Split

1. Pops a [Value](#value) `value` from the Entry stack.
2. Pops an int64 `newamount` from the data stack.
3. If `newamount` is greater than or equal to `value.amount`, fail execution.
4. Pushes a new Value with amount `value.amount - newamount` and assetID `value.assetID`.
5. Pushes a new Value with amount `newamount` and assetID `value.assetID`.


### Retire

1. Pops an item `value` of type [Value](#value) or [Proven Value](#proven-value) from the Entry stack.
2. If `value` is a plaintext [Value](#value), compute a corresponding non-blinded value commitment.
3. Pushes a [Retirement](#retirement) `r` to the Effect stack with `r.value` set to the value commitment.


### WrapValue

Converts plaintext value to a value commitment and emits a valid Asset Commitment usable in asset range proofs.

1. Pops a plaintext [Value](#value) item from the Entry stack.
2. Computes non-blinded [asset commitment](#asset-commitment) from the plaintext asset ID and pushes it to the Entry stack.
3. Computes non-blinded [value commitment](#value-commitment) from the plaintext amount and asset ID and pushes it to the Entry stack.


### MergeConfidential

1. Pops two items of type [Value](#value), [Proven Value](#proven-value) or [Unproven Value](#unproven-value) `a` and `b` from the [Entry stack](#entry-stack).
2. For each item of type [Value](#value) (if any) to the [Proven Value](#proven-value) with a corresponding [non-blinded value commitment](ca.md#create-nonblinded-value-commitment) based on plaintext `amount` and `assetID`.
3. Computes new [Unproven Value](#proven-value) `c` with `valuecommitment` equal to `a.valuecommitment + b.valuecommitment`.
4. Pushes unproven value `c` to the Entry stack.

Note: merging two proven values may merge two distinct asset IDs producing an unprovable value which must be correctly split and range-proved.


### SplitConfidential

1. Pops an item `value` of type [Value](#value), [Proven Value](#proven-value) or [Unproven Value](#unproven-value) from the Entry stack.
2. Pops a [Value Commitment](#value-commitment) `vc` from the Entry stack.
3. Pushes an [Unproven Value](#unproven-value) with `valuecommitment` equal to `vc`.
4. Pushes an [Unproven Value](#unproven-value) with `valuecommitment` equal to `value.valuecommitment - vc`.


### ProveAssetRange

This opcode proves that a given [Asset Commitment](#asset-commitment) belongs to a set of verified asset commitments.

1. Pops a string `ringsig` of `32*(n+1)` bytes from the data stack.
2. Pops string `program` from the data stack.
3. Pops [Asset Commitment](#asset-commitment) `ac` from the data stack.
4. Fail execution if the length of `ringsig` is less than 64 bytes, or not a whole number of 32 bytes.
5. Calculates `n = len(ringsig)/32 - 1`.
6. Peeks at `n` items `{prevac[i]}` of type [Asset Commitment](#asset-commitment) that should be located at the top of the Entry stack.
7. Constructs a [Confidential Asset Range Proof](ca.md#confidential-asset-range-proof) and verifies it using:
  * `ac` as a target asset ID commitment
  * `n` `prevac[i]` as previous asset ID commitments
  * `program` as the message signed by the range proof
8. Fails execution if verification of ARP fails.
9. Pushes asset commitment `ac` to the Entry stack.
10. Executes `program` via [command](#command) instruction.


### DropAssetCommitment

1. Pops an [Asset Commitment](#asset-commitment) `ac` from the Entry stack.
2. Fails if top element is not an asset commitment.

Note: in principle, proven asset ID commitments on Entry stack do not have to be specially consumed (like values), and the same commitments are reused in multiple ARPs. So to satisfy the requirement of a clean Entry stack and avoid unnecesary duplication of asset commitments, in the end of transaction, this opcode allows cleaning up all remaining asset commitments.


### ProveAssetID

This opcode proves that a given cleartext asset ID is stored within a given [Asset Commitment](#asset-commitment).

1. Pops string `assetproof` from the data stack.
2. Pops string `program` from the data stack.
3. Peeks at string `assetID` on the top the data stack.
4. Peeks at [Asset Commitment](#asset-commitment) `ac` below `assetID` on the data stack.
5. [Verifies](ca.md#validate-asset-id-proof) the `assetproof` with the given `assetID` and commitment `ac`.
6. Executes `program` via [command](#command) instruction.


### ProveAmount

1. Pops string `amountproof` from the data stack.
2. Pops string `program` from the data stack.
3. Peeks at integer `amount` on the top the data stack.
4. Peeks at [Value Commitment](#value-commitment) `vc` below `amount` on the data stack.
5. Peeks at [Asset Commitment](#asset-commitment) `ac` below `vc` on the data stack.
6. [Verifies](ca.md#validate-amount-proof) the `amountproof` string with the given `amount` and commitments `vc` and `ac`.
7. Executes `program` via [command](#command) instruction.


### ProveValueRange

1. Pops a string `valuerangeproof`.
2. Pops string `program` from the data stack.
3. Pops an item `value` of type [Unproven Value](#unproven-value) from the Entry stack.
4. Peeks at [Asset Commitment](#asset-commitment) `ac` at the top of the Entry stack.
5. Verifies `valuerangeproof` with the given `value.valuecommitment`, `ac.assetcommitment` and `program` as a custom message.
6. Pushes a new [Proven Value](#proven-value) to the Entry stack with `newvalue.valuecommitment` set to `value.valuecommitment` and `newvalue.assetcommitment` set to the `ac.assetcommitment`.
7. Executes `program` via [command](#command) instruction.


### IssuanceCandidate

1. Pops a Public Key `issuancekey` from the data stack.
2. Peeks at the top item on the Command stack, `p`.
3. Computes the [ID](#item-ids) `assetid` of an [asset definition](#asset-definition) tuple with `issuanceprogram` set to `p.program`.
4. Pushes an [Issuance Candidate](#issuance-candidate) to Entry stack with the `assetid` and `issuancekey`.

TBD: this is incompatible with existing asset IDs. We need either support for legacy asset definitions, or another opcode `LegacyIssuanceCandidate` to create ICs from legacy asset ids.

### IssueConfidential

1. Pops from data stack (in order):
  * [Value Commitment](#value-commitment) `vc`
  * [Asset Commitment](#asset-commitment) `ac`
  * string `program`
  * confidential IARP with `n` ring signature items
2. Pops `n` [Issuance Candidate](#issuance-candidate) items from Entry stack (`n` is the number of items in ring signature).
3. Verifies IARP using `ac`, `n` issuance candidates and a program as an IARP’s message.
4. Pushes asset commitment `ac` to the Entry stack.
5. Pushes new [Unproven Value](#unproven-value) with `valuecommitment` set to `vc` to the Entry stack.
6. Executes `program` via [command](#command) instruction.

Note: `IssueConfidential` authorized issuance of a certain asset commitment (within a given set of candidates) and a given value commitment. However, the value commitment must additionally be proven to be in range before it can be used. Program `program` allows issuer to commit to that value commitment, if needed.


## Anchor operations

### Nonce

1. Pops an int64 `min` from the data stack.
2. Pops an int64 `max` from the data stack.
3. Pops a string `blockchainid` from the data stack.
4. Peeks at the top item on the Command stack, `p`.
5. Constructs a [Nonce](#nonce) `nonce` with:
  * `nonce.program` equal to `p.program`,
  * `nonce.mintime` equal to `min`,
  * `nonce.maxtime` equal to `max`.
  * `nonce.blockchainid` equal to `blockchainid`.
6. Pushes `nonce` to the Effect stack.
7. Pushes an [anchor](#anchor) to the Entry stack with `value` equal to the [ID](#item-ids) of `nonce`.
8. Pushes a [Mintime](#mintime) to the Effect stack with `mintime` equal to `nonce.mintime`.
9. Pushes a [Maxtime](#maxtime) to the Effect stack with `maxtime` equal to `nonce.maxtime`.

### Reanchor

1. Pops an [anchor](#anchor) `anchor` from the Entry stack.
2. Compute the [ID](#item-ids) of `anchor`, `anchorid`.
3. Pushes a new anchor `newanchor` to the Entry stack, with `newanchor.value` set to `anchorid`.

### SplitAnchor

1. Pops an [anchor](#anchor) `anchor` from the Entry stack.
2. Compute the [ID](#item-ids) of `anchor`, `anchorid`.
3. Pushes a new anchor `newanchor01` to the Entry stack, with `newanchor.value` set to `sha3(0x01 || anchorid)`.
4. Pushes a new anchor `newanchor00` to the Entry stack, with `newanchor.value` set to `sha3(0x00 || anchorid)`.

### AnchorTransaction

Moves an [anchor](#anchor) `anchor` from the Entry stack to the Effect stack.



## Time operations

### Before

1. Pops an int64 `max` from the data stack.
2. Pushes a [Maxtime](#maxtime) to the [Effect stack](#Effect-stack) with `maxtime` equal to `max`.

### After

1. Pops an int64 `min` from the stack.
2. Pushes a [Mintime](#mintime) to the [Effect stack](#Effect-stack) with `mintime` equal to `min`.



## Conversion operations

### Summarize

1. Fails if transaction was already summarized.
2. Hashes encoded items on Effect stack from bottom to the top (see [Encode](#encode) instructions) using SHA3-256:

        h = SHA3-256(encode(item1) || encode(item2) || ... || encode(topitem))

3. Creates a tuple of type [Transaction Summary](#transaction-summary) `summary` with:
  * `version` equal to version specified in [Transaction](#transaction) tuple.
  * `runlimit` equal to runlimit specified in [Transaction](#transaction) tuple.
  * `effecthash` equal to `h`.
4. Pushes `summary` to the Effect stack.

Note: hashed items are unambiguously encoded, so the `effecthash` is equivalent to the hash of the items’ IDs, but avoid unnecessary memory and CPU overhead for multiple hash instances.

### UnlockLegacy

1. Pops a tuple of type [Legacy Output](#legacy-output) `legacy` from the data stack.
2. Computes legacy Output ID. TBD: specifics
3. Pushes an [Input](#input) to the Effect stack with `contractid` equal to the legacy output ID.
4. Constructs a tuple `a` of type [Anchor](#anchor) with `a.value` equal to the legacy output ID.
5. Pushes `a` to the Entry stack.
6. Constructs [Value](#value) tuple with the amount and asset ID specified in the legacy output, and pushes it to the Entry stack.
7. Instantiates legacy [VM1](vm1.md) with the following context:
  * TBD
  * TBD
  * TBD: need to defer this until txid is computed via `summarize`
8. TBD Alternatively: parse and translate the old-style program `legacy.program`, which must be a specific format, into a new one `newprogram`.
9. Defers execution of the legacy program. (TBD)

### IssueLegacy

TBD: Need compatibility layer to issue legacy asset IDs: specify the context for VM1 based on txvm tx.

### LegacyIssuanceCandidate

TBD: Need compatibility layer to use legacy asset IDs in the Issuance Candidates: also, specify necessary context for VM1 based on txvm tx.


### Extend

TBD: review this

1. Fails if the `extension` flag is `false`.
2. Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). 
3. Pops an integer `depth` from the data stack.
4. Pops an item, `extension`, from the data stack.
5. On the stack identified by `stackid`, takes the `n`th tuple and replaces it with a copy of that tuple with one additional field added, containing `extension`.

(TBD: be precise about how cost accounting works for this. Do we even still need this and in what form?)


### Extension opcodes

All non-assigned opcodes are NOPs (no effect).

Execution of a NOP fails the VM execution if the `extension` flag is `false`.

Have no effect when executed.


## Encoding opcodes

### Encode

Pops an item from the data stack. Pushes a string to the data stack which, if executed, would push that item to the data stack.

* **Strings** are encoded as a [Pushdata](#Pushdata) instruction which would push that string to the data stack.
* **Integers** in range 0..32 (inclusive) are encoded as the appropriate [small integer](#small-integer) opcode.
* **Other integers** (above 32 or negative) are encoded as [Pushdata](#Pushdata) instructions that would push the integer serialized as a [varint](#varint), followed by an [int64](#int64) instruction.
* **Tuples** are encoded recursively as the encoding of each item in the tuple in reverse order, followed by the encoding of `len` where `len` is the length of the tuple, followed by the [tuple](#tuple) instruction.

### Int64

1. Pops a string `a` from the stack,
2. decodes it as a [signed varint](#varint),
3. pushes the result to the data stack as an Int64.

Fails execution when:

* `a` is not a valid varint encoding of an integer,
* or the decoded `a` is greater than or equal to `2^63`.

### Small integers

TODO: Descriptions of opcodes that push the numbers 0-32 to the stack.

### Pushdata

[TBD: use Keith's method for this]




## Examples

### Normal transaction

TODO: fix now that Value, Anchor, and Condition stacks are merged

    {"anchor", "anchorvalue1..."} {{"value", 5, "assetid1..."}} 1 [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey1..." checksig verify] defer] command
    {"anchor", "anchorvalue2..."} {{"value", 10, "assetid1..."}} 1 [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey2..." checksig verify] defer] command
    {"anchor", "anchorvalue3..."} {{"value", 15, "assetid2..."}} 1 [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey3..." checksig verify] defer] command
    {"anchor", "anchorvalue4..."} {{"value", 20, "assetid2..."}} 1 [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey4..." checksig verify] defer] command
    merge
    2 valuestack roll
    2 valuestack roll
    merge
    6 split
    [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey5..." checksig verify] defer] lock
    [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey6..." checksig verify] defer] lock
    18 split
    [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey7..." checksig verify] defer] lock
    [jumpif:$unlock lock 1 jumpif:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey8..." checksig verify] defer] lock
    summarize
    "sig4..." satisfy
    "sig3..." satisfy
    "sig2..." satisfy
    "sig1..." satisfy


### Multi-asset contract


    // 5 of assetID1 and 10 of assetID2 are on the Entry stack
    [
        {"anchor", "anchorvalue1..."}
        {"value",  5, "assetid1..."}
      unlock
        {"anchor", "anchorvalue2..."}
        {"value",  10, "assetid2..."}
      unlock
      ["txvm" txstack peek encode cat sha3 "pubkey..." checksig verify] defer
    ] 0 datastack peek lock lock


### Issuance program signing transaction:

    [issue ["txvm" txstack peek encode cat sha3 "pubkey..." checksig verify] defer]

Usage (to issue 5 units):

    <sig> 5 [issue ["txvm" txstack peek encode cat sha3 "pubkey..." checksig verify] defer] command


### Issuance program signing anchor:

    [0 anchorstack peek]


### Maximally flexible issuance program

    [nonce amount [issue ] command

Maximally flexible issuance program:

    [nonce]
