# TxVM

This is the specification for txvm, which combines a representation for blockchain transactions with the rules for ensuring their validity.

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

Under txvm, these functions are combined in such a way that an executable program string is both the transaction’s representation and the proof of its validity.

When the virtual machine executes a txvm program, it accumulates different types of data on different stacks. This data corresponds to the information exposed in earlier versions of the transaction data structure: inputs, outputs, time constraints, nonces, and so on. Under txvm, that information is _only_ available as a result of executing the program, and the program only completes without error if the transaction is well-formed (i.e., its inputs and outputs balance, prevout control programs are correctly satisfied, etc). No separate validation steps are required.

The pieces of transaction information - the inputs, outputs, etc. - that are produced during txvm execution are also _consumed_ in order to produce the transaction summary, which is the sole output of a successful txvm program. To capture pieces of transaction information for purposes other than validation, txvm implementations can and should provide callback hooks for inspecting and copying data from the various stacks at key points during execution.


## TxVM operation

Validation of the transaction happens in a context of a validating a block of transactions. Large part of that validation is handled by the TxVM logic with a few validation rules outside of it.

### Transaction version

TBD: how transaction version is specified and how `extension` flag is set.

Sketch:

1. New txvm txs will have version 2 to avoid confusion with txv1 (they have incompatible format, but still). 
2. Version 1 is prohibited in txvm.
3. Tx version can be unknown (>2) only if allowed by outer context (e.g. block version is unknown)
4. If tx version is unknown (>2) extension flag is set to true to allow NOPs and extends.

TBD: should we specify txversion inside the bytecode or in the container? E.g. we could have "transaction" tuple:

    {
      type:    "tx", 
      version: 2, 
      program: "...txvm bytecode..."
    }

### VM Execution

1. The VM is initialized with all stacks empty.
2. TXVM bytecode is being executed.
3. When the program counter is equal to the length of the program, execution is complete. 
4. The top item of the [Effect stack](#Effect) must be a [Transaction Summary](#transaction-summary).
5. There must be no other Transaction Summaries in the Effect stack, otherwise execution fails.
6. There must be at least one [anchor](#anchor) in the Effect stack. 
7. The Entry stack must be empty.

### Post-execution

If execution and all the required checks do not fail, Effect stack is introspected and blockchain state is updated:

1. [Transaction ID](#transaction-id) is committed to the block as ID of the Transaction Summary.
2. For each [Input](#input), its `contractid` is removed from the UTXO set.
3. For each [Output](#output), its `contractid` is added to the UTXO set.
4. Remove all outdated nonces from Nonce set (based on block's timestamp)
5. For each [Nonce](#nonce), add it's ID to the Nonce set.
6. TBD: records?


### Runlimit

The VM is initialized with a set runlimit. Each instruction reduces that number. If the runlimit goes below zero while the program counter is less than the length of the program, execution fails.

1. Each instruction costs `1`.
2. Each instruction that pushes an item to the data stack, including as the result of an operation (such as `add`, `cat`, `merge`, `field`, and `untuple`), costs an amount based on the type and size of that data:
  1. Each string that is pushed to the stack costs `1 + len`, where `len` is the length of that string in bytes.
  2. Each number that is pushed to the stack costs `1`.
  3. Each tuple that is pushed to the stack costs `1 + len`, where `len` is the length of that tuple.
3. Each instruction that pushes an item to any stack other than the data or alt stack costs `256` for each item so pushed.
4. Each `checksig` and `pointmul` instruction costs `1024`. [TBD: estimate the actual cost of these instruction relative to the other instructions].
5. Each `roll`, `bury`, or `reverse` instruction costs `n`, where `n` is the `n` argument to that operation.

TODO: suggestion - specify runlimit in the transaction structure. Consume that limit from the one declared in the block. Federation chooses appropriate limit and signs over it, preventing DoS (because tx ID is computed only via execution of txvm).


## Compatibility

TxVM transactions are not compatible with version 1 transactions. However, they allow interacting with pre-existing blockchain state: nonces, outputs and asset IDs.

### Spending legacy outputs

TBD: overview of the upgrade opcode

### Issuance of legacy Asset ID

TBD: Need compatibility layer to issue legacy asset IDs: specify the context for VM1 based on txvm tx.

### Confidential issuance of legacy Asset IDs

TBD: Need compatibility layer to use legacy asset IDs in the Issuance Candidates: also, specify necessary context for VM1 based on txvm tx.

### Soft-fork and hard-fork upgrades to TxVM

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
4. `genesisblockid`, a string

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

### Command

0. `type`, a string, "command"
1. `program`, a string

### Transaction Summary

0. `type`, a string, "transactionSummary"
1. `effecthash`, a 32-byte hash of all the effect entries

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

Items on the Command stack are [Commands](#command).

### Effect stack

Items on the Effect stack are [Inputs](#input), [Outputs](#output), [Reads](#read), [Nonces](#nonce), [Retirements](#retirement).

### Issuance candidates stack

Items on the Issuance candidates stack are [Issuance Candidates](#issuance-candidates).






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

Pops two integers `a` and `b` from the stack, shifts `a` to the right by `b` bits.

TODO: clarify behavior. 

### GreaterThan

Pops two numbers `a` and `b` from the stack. If `a` is greater than `b`, pushes `true` to the stack. Otherwise, pushes `false`.



## String operations

### Cat

Pops two strings, `a`, then `b`, from the stack, concatenates them, and pushes the result, `a || b` to the stack.

### Slice

Pops two integers, `start`, then `end`, from the stack. Pops a string `str` from the stack. Pushes the string `str[start:end]` (with the first character being the one at index `start`, and the second character being the one at index `end`). Fails if `end` is less than `start`, if `start` is less than 0, or if `end` is greater than the length of `str`.



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

1. Pops a string `program` from the data stack. 
2. Constructs a tuple `command` of type [Command](#command) with `program` equal to `program`. 
3. Pushes `command` to the Command stack. 
4. Executes `command.program`. 
5. Pops a [Command](#command) from the Command stack. 

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
2. Peeks at the top item on the Command stack,`command`. 
3. If `record.commandprogram` is not equal to `command.program`, fails execution.


### Complete

1. Peeks at the top item on the Record stack, `record`.
2. Peeks at the top item on the Command stack, `command`.
3. If `record.commandprogram` is not equal to `command.program`, fails execution. 
4. Moves `record` to the Effect stack.



## Contract operations

### Unlock 

1. Pops an item `value` of type [Value](#value) or [Proven Value](#proven-value) from the data stack. 
2. Pops an item of type [Anchor](#anchor) from the data stack. 
3. Peeks at the top [Command](#command) `command` on the Command stack.
4. Constructs a tuple `input` of type [Contract](#contract), with:
  * `input.program` equal to `command.program`, 
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
3. Peeks at the top [Command](#command) `command` on the Command stack.
4. Constructs a tuple `contract` of type [Contract](#contract), with:
  * `contract.program` equal to `command.program`, 
  * `contract.anchor` equal to `anchor.value`, and 
  * `contract.value` equal to `value`. 
5. Computes the [ID](#item-ids) `contractid` of `input`. 
6. Pushes a [Read](#read) to the Effect stack with `contractid` equal to `contractid`.

### Lock

1. Pops an item of type [Value](#Value) or [Proven Value](#proven-value), `value`, from the Entry stack.
2. Pops an [anchor](#anchor) `anchor` from the Entry stack.
3. Peeks at the top [Command](#command) `command` on the Command stack.
4. Constructs a tuple `contract` of type [Contract](#contract), with:
  * `contract.program` equal to `command.program`,
  * `contract.anchor` equal to `anchor`,
  * `contract.value` equal to `value`.
5. Computes the [ID](#item-id) `contractid` of `contract`.
6. Pushes an [Output](#output) to the Effect stack with `contractid` equal to `contractid`.


## Value operations

### Issue

1. Pops an int64 `amount` from the data stack. 
2. Peeks at the top item on the Command stack, `command`. 
3. Computes the [ID](#item-ids) `assetid` of an [asset definition](#asset-definition) tuple with `issuanceprogram` set to `command.program`. 
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


### MergeConfidential

1. Pops two items of type [Value](#value), [Proven Value](#proven-value) or [Unproven Value](#unproven-value) `a` and `b` from the [Entry stack](#entry-stack).
2. Converts each item of type [Value](#value) (if any) to the [Proven Value](#proven-value) with a corresponding [non-blinded value commitment](ca.md#create-nonblinded-value-commitment) based on plaintext `amount` and `assetID`.
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
2. Peeks at the top item on the Command stack, `command`. 
3. Computes the [ID](#item-ids) `assetid` of an [asset definition](#asset-definition) tuple with `issuanceprogram` set to `command.program`.
4. Pushes an [Issuance Candidate](#issuance-candidate) to Entry stack with the `assetid` and `issuancekey`.

TBD: this is incompatible with existing asset IDs. We need either support for legacy asset definitions, or another opcode `LegacyIssuanceCandidate` to create ICs from legacy asset ids.

### ConfidentialIssue

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

Note: `ConfidentialIssue` authorized issuance of a certain asset commitment (within a given set of candidates) and a given value commitment. However, the value commitment must additionally be proven to be in range before it can be used. Program `program` allows issuer to commit to that value commitment, if needed.


## Anchor operations

### Nonce

1. Pops an int64 `min` from the data stack. 
2. Pops an int64 `max` from the data stack. 
3. Pops a string `blockchainid`.
4. Peeks at the top item on the Command stack, `command`.
5. Verifies that `blockchainid` is equal to the blockchain ID. 
6. Constructs a [Nonce](#nonce) `nonce` with:
  * `nonce.program` equal to `command.program`, 
  * `nonce.mintime` equal to `min`, 
  * `nonce.maxtime` equal to `max`. 
7. Pushes `nonce` to the Effect stack. 
8. Pushes an [anchor](#anchor) to the Entry stack with `value` equal to the [ID](#item-ids) of `nonce`. 
9. Pushes a [Mintime](#mintime) to the Effect stack with `mintime` equal to `nonce.mintime`. 
10. Pushes a [Maxtime](#maxtime) to the Effect stack with `maxtime` equal to `nonce.maxtime`.

### Reanchor

1. Pops an [anchor](#anchor) `anchor` from the Entry stack. 
2. Compute the [ID](#item-ids) of `anchor`, `anchorid`. 
3. Pushes a new anchor `newanchor`, with `newanchor.value` set to `anchorid`.

### Splitanchor

1. Pops an [anchor](#anchor) `anchor` from the Entry stack. 
2. Compute the [ID](#item-ids) of `anchor`, `anchorid`. 
3. Pushes a new anchor `newanchor01`, with `newanchor.value` set to `sha3("01" || anchorid)`. Pushes a new anchor `newanchor00`, with `newanchor.value` set to `sha3("00" || anchorid)`.

### Anchortransaction

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

1. Hashes encoded items on Effect stack from bottom to the top (see [Encode](#encode) instructions) using SHA3-256:

        h = SHA3-256(encode(item1) || encode(item2) || ... || encode(topitem))

2. Creates a tuple of type [Transaction Summary](#transaction-summary) `summary` with `effecthash` equal to `h`. 
3. Pushes `summary` to the Effect stack.

Note: hashed items are unambiguously encoded, so the `effecthash` is equivalent to the hash of the items’ IDs, but avoid unnecessary memory and CPU overhead for multiple hash instances.

### Migrate

TODO: we need to convert legacy output to `LegacyInput` so we can put it on the Effect stack and remove the corresponding outputid from UTXO set. And then unlock the value.

Pops a tuple of type [legacy output](#legacy-output) `legacy` from the data stack. Pushes it to the Effect stack. Pushes an [anchor](#anchor) to the Entry stack with `value` set to the old-style ID (TBD) of `legacy`.

[TBD: parse and translate the old-style program `legacy.program`, which must be a specific format, into a new one `newprogram`.]

Pushes a [Value](#value) with amount `legacy.amount` and asset ID `legacy.assetID` to the Entry stack.

Executes `newprogram`.

### Extend

Fails if the extension flag is not set. (TBD: clarify)

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Pops an integer `depth` from the data stack. Pops an item, `extension`, from the data stack. On the stack identified by `stackid`, takes the `n`th tuple and replaces it with a copy of that tuple with one additional field added, containing `extension`.

(TBD: be precise about how cost accounting works for this. Do we even still need this and in what form?)


## Extension opcodes

### NOPs 0–9

Fails if the extension flag is not set. (TBD: clarify)

Have no effect when executed.

### Reserved

TODO: do we really need reserved opcodes? All NOPs are prohibited w/o "extensible" flag turned on (that is for unknown tx versions).

Causes the VM to halt and fail.



## Encoding opcodes

### Encode

Pops an item from the data stack. Pushes a string to the data stack which, if executed, would push that item to the data stack.

* **Strings** are encoded as a [Pushdata](#Pushdata) instruction which would push that string to the data stack. 
* **Integers** in range 0..32 (inclusive) are encoded as the appropriate [small integer](#small-integer) opcode. 
* **Other integers** (above 32 or negative) are encoded as [Pushdata](#Pushdata) instructions that would push the integer serialized as a [varint](#varint), followed by an [int64](#int64) instruction. 
* **Tuples** are encoded as a sequence of [Pushdata](#Pushdata) instructions that would push each of the items in the tuple in reverse order, followed by the instruction given by `encode(len)` where `len` is the length of the tuple, followed by the [tuple](#tuple).

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
