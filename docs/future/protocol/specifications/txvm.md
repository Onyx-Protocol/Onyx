# TxVM

This is the specification for txvm, which combines a representation for blockchain transactions with the rules for ensuring their validity.

## Motivation

Earlier versions of Chain Core represented transactions with a static data structure, exposing the pieces of information needed to test the transaction’s validity. A separate set of validation rules could be applied to that information to get a true/false result.

Under txvm, these functions are combined in such a way that an executable program string is both the transaction’s representation and the proof of its validity.

When the virtual machine executes a txvm program, it accumulates different types of data on different stacks. This data corresponds to the information exposed in earlier versions of the transaction data structure: inputs, outputs, time constraints, nonces, and so on. Under txvm, that information is _only_ available as a result of executing the program, and the program only completes without error if the transaction is well-formed (i.e., its inputs and outputs balance, prevout control programs are correctly satisfied, etc). No separate validation steps are required.

The pieces of transaction information - the inputs, outputs, etc. - that are produced during txvm execution are also _consumed_ in order to produce the transaction summary, which is the sole output of a successful txvm program. To capture pieces of transaction information for purposes other than validation, txvm implementations can and should provide callback hooks for inspecting and copying data from the various stacks at key points during execution.

# VM Execution

The VM is initialized with all stacks empty.

When the program counter is equal to the length of the program, execution is complete. The top item of the [Effect stack](#Effect) must be a [Transaction ID](#transaction-id), and there must be no other Transaction IDs in the Effect stack. There must be at least one [anchor](#anchor) in the Effect stack. The Entry stack must be empty.

# Stacks

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

### Tuple

An immutable collection of items of any type.

There are several named types of tuples.

#### Value

0. `type`, a string, "value"
1. `amount`, an int64
2. `assetID`, a string

#### Value Commitment

0. `type`, a string, "valuecommitment"
1. `rawvaluecommitment`, a raw [value commitment](ca.md#value-commitment) as described in [CA](ca.md) specification.

#### Asset Commitment

0. `type`, a string, "assetcommitment"
1. `rawassetcommitment`, a raw [asset ID commitment](ca.md#asset-id-commitment) as described in [CA](ca.md) specification.

#### Asset Range Proof

TBD

#### Value Range Proof

TBD

#### Unproven Value

0. `type`, a string, "unprovenvalue"
1. `valuecommitment`, a [value commitment](#value-commitment)

#### Proven Value

0. `type`, a string, "provenvalue"
1. `valuecommitment`, a [value commitment](#value-commitment)
2. `assetcommitment`, an [asset commitment](#asset-commitment)

## Record type

0. `type`, a string, "record"
1. `commandprogram`, a string
2. `data`, an item

#### Input

0. `type`, a string, "input"
1. `contractid`, a string

#### Output

0. `type`, a string, "output"
1. `contractid`, a string

#### Read

0. `type`, a string, "read"
1. `contractid`, a string

#### Contract

0. `type`, a string, "contract"
1. `values`, a tuple of either [values](#values) or [proven values](#proven-values)
2. `program`, a [Program](#program)
3. `anchor`, a string

#### Program

0. `type`, a string, "program"
1. `program`, a string

#### Nonce

0. `type`, a string, "nonce"
1. `program`, a string
2. `mintime`, an int64
3. `maxtime`, an int64
4. `genesisblockid`, a string

#### Anchor

0. `type`, a string, "anchor"
1. `value`, a string

#### Retirement

0. `type`, a string, "retirement"
1. `value`, a [value commitment](#value-commitment)

#### Asset Definition

0. `type`, a string, "assetdefinition"
1. `issuanceprogram`, a [Program](#program)

#### Issuance Candidate

0. `type`, a string, "issuancecandidate"
1. `assetID`, a string
2. `issuanceKey`, a [Public Key](#public-key)

#### Maxtime

0. `type`, a string, "maxtime"
1. `maxtime`, an int64

#### Mintime

0. `type`, a string, "mintime"
1. `mintime`, an int64

### Annotation

0. `type`, a string, "annotation"
1. `data`, a string

### Command

0. `type`, a string, "command"
1. `program`, a string

#### Transaction Summary

0. `type`, a string, "transactionSummary"
1. `effectids`, a tuple of items

#### Transaction ID

0. `type`, a string, "transactionID"
1. `transactionid`, a string

### Legacy Output

0. `sourceID`, a 32-byte ID
1. `assetID`, a 32-byte asset ID
2. `amount`, an int64
3. `index`, an int64
4. `program`, a string
5. `data`, a string

## Item IDs

The ID of an item is the SHA3 hash of `"txvm" || encode(item)`, where `encode` is the [encode](#encode) operation.

## Stack identifiers

TBD: starting with 0 here can cause off-by-one errors - when formatted in markdown the list will start with 1.
I suggest starting with 1 to avoid this, or use a table with a column for indices.

0. Data stack
1. Alt stack
2. Entry stack
3. Command stack
4. Effect stack
5. Issuance candidates stack

## Data stack

Items on the data stack can be int64s, strings, or tuples.

### Alt stack

Items on the alt stack have the same types as items on the data stack. The alt stack starts out empty. Items can be moved from the data stack to the alt stack with the [toaltstack](#toaltstack) instruction, and from the alt stack to the data stack with the [fromaltstack](#fromaltstack).

### Entry stack

Items on the Entry stack are [Values](#value)

### Effect stack

Items on the Effect stack are [Inputs](#input), [Outputs](#output), [Reads](#read), [Nonces](#nonce), [Retirements](#retirement).

### Issuance candidates stack

Items on the Issuance candidates stack are [Issuance Candidates](#issuance-candidates).

# Encoding formats

## Varint

TODO: Describe rules for encoding and decoding unsigned varints.

## Point

See [Point](ca.md#point) definition in Confidential Assets specification.

## Point Pair

See [Point Pair](ca.md#point-pair) definition in Confidential Assets specification.

# Runlimit

The VM is initialized with a set runlimit. Each instruction reduces that number. If the runlimit goes below zero while the program counter is less than the length of the program, execution fails.

1. Each instruction costs `1`.
2. Each instruction that pushes an item to the data stack, including as the result of an operation (such as `add`, `cat`, `merge`, `field`, and `untuple`), costs an amount based on the type and size of that data:
  1. Each string that is pushed to the stack costs `1 + len`, where `len` is the length of that string in bytes.
  2. Each number that is pushed to the stack costs `1`.
  3. Each tuple that is pushed to the stack costs `1 + len`, where `len` is the length of that tuple.
3. Each instruction that pushes an item to any stack other than the data or alt stack costs `256` for each item so pushed.
4. Each `checksig` and `pointmul` instruction costs `1024`. [TBD: estimate the actual cost of these instruction relative to the other instructions].
5. Each `roll`, `bury`, or `reverse` instruction costs `n`, where `n` is the `n` argument to that operation.

# Operations

## Control flow operations

### Fail

Halts VM execution, returning false.

### PC

Pushes the current program counter (after incrementing for this instruction) to the data stack.

### JumpIf

Pops an integer `destination`, then a boolean `cond` from the data stack. If `cond` is false, do nothing. If `cond` is true, set program counter to `destination`. Fail if `destination` is negative, if `destination` is greater than the length of the current program.

## Stack operations 

### Roll

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops another integer `n` from the data stack. Fails if `stackid` refers to the Command stack or the Effect stack. 

On the stack identified by `stackid`, moves the `n`th item from the top from its current position to the top of the stack.

Fails if `stackid` does not correspond to a valid stack, or if the stack has fewer than `n + 1` items.

### Bury

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops a number `n` from the data stack. Fails if `stackid` refers to the Command stack or the Effect stack. 

On the stack identified by `stackid`, moves the top item and inserts it at the `n`th-from-top position.

### Reverse

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops a number `n` from the data stack. Fails if `stackid` refers to the Command stack or the Effect stack. On the stack identified by `stackid`, removes the top `n` items and inserts them back into the same stack in reverse order.

### Depth

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Counts the number of items on the stack identified by `stackid`, and pushes it to the data stack.

### Peek

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Pops an integer `n` from the data stack. Looks at the `n`th item of the stack identified by `stackid`, and pushes a copy of it to the data stack.

## Data stack operations

### Equal

Pops two items `val1` and `val2` from the data stack. If they have different types, or if either is a tuple, fails execution. If they have the same type: if they are equal, pushes `true` to the stack; otherwise, pushes `false` to the stack.

### Type

Looks at the top item on the data stack. Pushes a number to the stack corresponding to that item's [type](#type).

### Len

Pops a string or tuple `val` from the data stack. If `val` is a tuple, pushes the number of fields in that tuple to the data stack. If `val` is a string, pushes the length of that string to the data stack. Fails if `val` is a number.

### Drop

Drops an item from the data stack.

### ToAlt

Pops an item from the data stack and pushes it to the alt stack.

### FromAlt

Pops an item from the alt stack and pushes it to the data stack.

## Tuple operations 

### Tuple

Pops an integer `len` from the data stack. Pops `len` items from the data stack and creates a tuple of length `len` on the data stack.

### Untuple

Pops a tuple `tuple` from the data stack. Pushes each of the fields in `tuple` to the stack in reverse order (so that the 0th item in the tuple ends up on top of the stack).

### Field

Pops an integer `i` from the top of the data stack, and pops a tuple `tuple`. Pushes the item in the `i`th field of `tuple` to the top of the data stack.

Fails if `i` is negative or greater than or equal to the number of fields in `tuple`.

## Boolean operations

### Not

Pops a boolean `p` from the stack. If `p` is `true`, pushes `false`. If `p` is `false`, pushes `true`.

### And

Pops two booleans `p` and `q` from the stack. If both `p` and `q` are true, pushes `true`. Otherwise, pushes `false`.

### Or

Pops two booleans `p` and `q` from the stack. If both `p` and `q` are false, pushes `false`. Otherwise, pushes `true`.

## Math operations

### Add

Pops two integers `a` and `b` from the stack, adds them, and pushes their sum `a + b` to the stack.

### Sub

Pops two integers `a` and `b` from the stack, subtracts the second frmo the first, and pushes their difference `a - b` to the stack.

### Mul

Pops two integers `a` and `b` from the stack, multiplies them, and pushes their product `a * b` to the stack.

### Div

Pops two integers `a` and `b` from the stack, divides them truncated toward 0, and pushes their quotient `a / b` to the stack.

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

Pops two strings, `a`, then `b`, from the stack, concatenates them, and pushes the result, `a ++ b` to the stack.

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

Pops a string `a` from the data stack. Performs a Sha2-256 hash on it, and pushes the result `sha256(a)` to the data stack.

### SHA3

Pops a string `a` from the data stack. Performs a Sha3-256 hash on it, and pushes the result `sha3-256(a)` to the data stack.

### CheckSig

Pops a string `pubKey`, a string `msg`, and a string `sig` from the data stack. Performs an Ed25519 signature check with `pubKey` as the public key, `msg` as the message, and `sig` as the signature. Pushes `true` to the data stack if the signature check succeeded, and `false` otherwise.

TODO: Should we switch order of `pubKey` and `msg`?

### PointAdd

Pops two strings `a` and `b` from the data stack, decodes each of them as [Ed25519 curve points](#ed25519-curve-points), performs an elliptic curve addition `a + b`, encodes the result as a string, and pushes it to the data stack. Fails if `a` and `b` are not valid curve points.

### PointSub

Pops two strings `a` and `b` from the data stack, decodes each of them as [Ed25519 curve points](#ed25519-curve-points), performs an elliptic curve subtraction `a - b`, encodes the result as a string, and pushes it to the data stack. Fails if `a` and `b` are not valid curve points.

### PointMul

Pops an integer `i` and a string `a` from the data stack, decodes `a` as an [Ed25519 curve points](#ed25519-curve-points), performs an elliptic curve scalar multiplication `i*a`, encodes the result as a string, and pushes it to the data stack. Fails if `a` is not a valid curve point.

## Annotation operations

### Annotate

Pops a string, `data`, from the data stack. Pushes an [Annotation](#annotation) with `data` of `data` to the Effect stack.

## Command operations

### Command

Pops a string `program` from the data stack. Constructs a tuple `command` of type [Command](#command) with `program` equal to `program`. Pushes `command` to the Command stack. Executes `command.program`. Pops a [Command](#command) from the Command stack.

## Condition operations

### Defer

Pops a [Program](#program) from the data stack and pushes it to the Entry stack.

### Satisfy

Pops a condition from the Entry stack and executes it using [command](#command).

## Record operations

### Create

Pops an item, `data`, from the data stack. Peeks at the top item on the Command stack,`command`. Pushes a Record to the Entry stack with `commandprogram` equal to `command.program` and `data` equal to `data`.

### Delete

Pops a Record, `record`, from the Record stack. Peeks at the top item on the Command stack,`command`. If `record.commandprogram` is not equal to `command.program`, fails execution.

### Complete

Peeks at the top item on the Record stack, `record`, and the top item on the Command stack, `command` If `record.commandprogram` is not equal to `command.program`. fails execution. Moves `record` to the Effect stack.

## Contract operations

### Unlock 

Pops an item `value` of type [Value](#value) or [Proven Value](#proven-value) from the data stack. Pops an item of type [Anchor](#anchor) from the data stack. Peeks at the top [Command](#command) `command` on the Command stack.

Constructs a tuple `input` of type [Contract](#contract), with `program` equal to `command.program`, `anchor` equal to `anchor`, and `value` equal to `value`. Computes the [ID](#item-ids) `contractid` of `input`. Pushes an [Input](#input) to the Effect stack with `contractid` equal to `contractid`.

If `value` is a [Proven Value](#proven-value), pushes `value.assetcommitment` to the Entry stack.

Constructs a tuple `anchor` of type [Anchor](#anchor) with `value` equal to `input.anchor`. Pushes `anchor` to the Entry stack. 

Pushes `value` to the Entry stack.

### Read

Pops an item `value` of type [Value](#value) or [Proven Value](#proven-value) from the data stack. Pops an item of type [Anchor](#anchor) from the data stack. Peeks at the top [Command](#command) `command` on the Command stack.

Constructs a tuple `contract` of type [Contract](#contract), with `program` equal to `command.program`, `anchor` equal to `anchor`, and `value` equal to `value`. Computes the [ID](#item-ids) `contractid` of `input`. Pushes a [Read](#read) to the Effect stack with `contractid` equal to `contractid`.

### Lock

Pops an item of type [Value](#Value) or [Proven Value](#proven-value), `value`, from the Entry stack. Pops an [anchor](#anchor) `anchor` from the Entry stack. Peeks at the top [Command](#command) `command` on the Command stack.

CConstructs a tuple `contract` of type [Contract](#contract), with `program` equal to `command.program`, `anchor` equal to `anchor`, and `value` equal to `value`. Computes the [ID](#item-id) `contractid` of `contract`. Pushes an [Output](#output) to the Effect stack with `contractid` equal to `contractid`.

## Value operations

### Issue

Pops an int64 `amount` from the data stack. Peeks at the top item on the Command stack, `command`. Computes the [ID](#item-ids) `assetid` of an [asset definition](#asset-definition) tuple with `issuanceprogram` set to `command.program`. Pushes a [value](#value) with amount `amount` and assetID `assetID`.

### Merge

Pops two [Values](#value) from the Entry stack. If their asset IDs are different, execution fails. Pushes a new [Value](#value) to the Entry stack, whose asset ID is the same as the popped values, and whose amount is the sum of the amounts of each of the popped values.

### Split

Pops a [Value](#value) `value` from the Entry stack. Pops an int64 `newamount` from the data stack. If `newamount` is greater than or equal to `value.amount`, fail execution. Pushes a new Value with amount `newamount - value.amount` and assetID `value.assetID`, then pushes a new Value with amount `newamount` and assetID `value.assetID`.

### Retire

Pops a [Value](#value) `value` or [Proven Value](#proven-value) from the Entry stack. Pushes a [Retirement](#retirement) to the Effect stack with `value` equal to `value`.

## Confidential value operations

### MergeConfidential

FIXME: VULNERABILITY: merging unprovable and proven values allows creating provable value. We should allow merging only proven values.

Pops two items of type [Proven Value](#proven-value) or [Unproven Value](#unproven-value) `value1` and `value2` from the [Entry stack](#value-stack).

Pushes an [Unproven Value](#unproven-value) with `valuecommitment` equal to `value1.valuecommitment + value2.valuecommitment` to the Entry stack.

### SplitConfidential

Pops an item `value` of type [Proven Value](#proven-value) or [Unproven Value](#unproven-value) from the Entry stack. Pops a [Value Commitment](#value-commitment) `newvaluecommitment` from the Entry stack. Pushes an [Unproven Value](#unproven-value) with `valuecommitment` equal to `newvaluecommitment`, then pushes an [Unproven Value](#unproven-value) with `valuecommitment` equal to `value.valuecommitment - newvaluecommitment`.

### ProveAssetCommitment

Pops an item `assetrangeproof` of type [Asset Range Proof](#asset-range-proof). Pops an item `assetcommitment` from the data stack of type [Asset Commitment](#asset-commitment). Pops an item 

Verifies `assetrangeproof` with ` assetcommitment` as the asset commitment. 

TBD: LINK THIS, ADD CANDIDATES, FIX TERMINOLOGY, AND ADD ANYTHING ELSE.

Pushes an `assetcommitment` to the Entry stack.

### ProveValue

Pops an item `valuerangeproof` of type [Value Range Proof](#value-range-proof) from the data stack. Pops an item `value` of type [Unproven Value](#unproven-value) from the Entry stack. Pops an item of type [Asset Commitment](#asset-commitment) `assetcommitment` from the Entry

Verifies `valuerangeproof` with ` value.valuecommitment` as the value commitment, and `assetcommitment` as the asset commitment. 

TBD: LINK THIS, FIX TERMINOLOGY, AND ADD ANYTHING ELSE.

Pushes a [Proven Value](#proven-value) to the Entry stack with `value.valuecommitment` as the `valuecommitment` and `assetcommitment` as the asset commitment.

### IssuanceCandidate

Pops a Public Key `issuancekey` from the data stack. Peeks at the top item on the Command stack, `command`. Computes the [ID](#item-ids) `assetid` of an [asset definition](#asset-definition) tuple with `issuanceprogram` set to `command.program`.

Pushes an [Issuance Candidate](#issuance-candidate) with `assetid` of `assetid` and `issuancekey` of `issuancekey`.

TBD: this is incompatible with existing asset IDs. We need either support for legacy asset definitions, or another opcode `LegacyIssuanceCandidate` to create ICs from legacy asset ids.

### IssueCA

(WIP. TBD: REVIEW AND REWRITE THIS; MAYBE SPLIT INTO MULTIPLE OPCODES)

Pops from data stack:

* AC
* VC
* iarp-condition
* list of candidate tuples (asset definition, issuance pubkey)
* IARP
* VRP

Verifies IARP using issuance pubkeys and signing over (AC,VC,iarp-condition)

Verifies VRP over (AC,VC,iarp-condition).

Pushes:

* AC and VC to PAC-stack and PVC-stack respectively. 
* `iarp-condition` to condition stack. 
* each `(assetdefinition, issuance pubkey)` to IC-stack.
* each `assetdefinition.issuanceprogram` to condition stack.

IC-stack is necessary so that `issuanceprogram` can verify that the correct issuance key is used.

## Anchor operations

### Nonce

Pops an int64 `min` from the data stack. Pops an int64 `max` from the data stack. Pops a string `blockchainid` Peeks at the top item on the Command stack, `command`.

Verifies that `blockchainid` is equal to the blockchain ID. Constructs a [Nonce](#nonce) `nonce` with `program` equal to `command.program`, `min` equal to `min`, and `max` equal to `max`. Pushes `nonce` to the Effect stack. Pushes an [anchor](#anchor) to the Entry stack with `value` equal to the [ID](#item-ids) of `nonce`. Pushes a [Mintime](#mintime) to the Effect stack with `mintime` equal to `mintime`. Pushes a [Maxtime](#maxtime) to the Effect stack with `maxtime` equal to `nonce.maxtime`.

### Reanchor

Pops an [anchor](#anchor) `anchor` from the Entry stack. Compute the [ID](#item-ids) of `anchor`, `anchorid`. Pushes a new anchor `newanchor`, with `newanchor.value` set to `anchorid`.

### Splitanchor

Pops an [anchor](#anchor) `anchor` from the Entry stack. Compute the [ID](#item-ids) of `anchor`, `anchorid`. Pushes a new anchor `newanchor01`, with `newanchor.value` set to `sha3("01" ++ anchorid)`. Pushes a new anchor `newanchor00`, with `newanchor.value` set to `sha3("00" ++ anchorid)`.

### Anchortransaction

Moves an [anchor](#anchor) `anchor` from the Entry stack to the Effect stack.

## Mintime and Maxtime operations

### Before

Pops an int64 `max` from the data stack. Pushes a [Maxtime](#maxtime) to the [Effect stack](#Effect-stack) with `maxtime` equal to `max`.

### After

Pops an int64 `min` from the stack. Pushes a [Mintime](#mintime) to the [Effect stack](#Effect-stack) with `mintime` equal to `min`.

### Summarize

Computes the ID of each item on the Effect stack. Creates a tuple of those IDs (with the first item first), `effectids`. Creates a tuple of type [Transaction Summary](#transaction-summary) `summary` with `effectids` equal to `effectids`. Computes the [ID](#item-id) `txid` of `summary`. Creates a tuple of type [Transaction ID](#transaction-id) on the Effect stack with `transactionid` equal to `transactionid`.

### Migrate

(TBD: update)

Pops a tuple of type [legacy output](#legacy-output) `legacy` from the data stack. Pushes it to the Effect stack. Pushes an [anchor](#anchor) to the Entry stack with `value` set to the old-style ID (TBD) of `legacy`.

[TBD: parse and translate the old-style program `legacy.program`, which must be a specific format, into a new one `newprogram`.]

Pushes a [Value](#value) with amount `legacy.amount` and asset ID `legacy.assetID` to the Entry stack.

Executes `newprogram`.

### Extend

Fails if the extension flag is not set. (TBD: clarify)

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Pops an integer `depth` from the data stack. Pops an item, `extension`, from the data stack. On the stack identified by `stackid`, takes the `n`th tuple and replaces it with a copy of that tuple with one additional field added, containing `extension`.

(TBD: be precise about how cost accounting works for this. Do we even still need this and in what form?)

### Blind

TBD

## Extension opcodes

### NOPs 0–9

Fails if the extension flag is not set. (TBD: clarify)

Have no effect when executed.

### Reserved

Causes the VM to halt and fail.

## Encoding opcodes

### Encode

Pops an item from the data stack. Pushes a string to the data stack which, if executed, would push that item to the data stack.

Strings are encoded as a [Pushdata](#Pushdata) instruction which would push that string to the data stack. Integers greater than or equal to 0 and less than or equal to 32 are encoded as the appropriate [small integer](#small-integer) opcode. Other integers are encoded as [Pushdata](#Pushdata) instructions that would push the integer serialized as a [varint](#varint), followed by an [int64](#int64) instruction. Tuples are encoded as a sequence of [Pushdata](#Pushdata) instructions that would push each of the items in the tuple in reverse order, followed by the instruction given by `encode(len)` where `len` is the length of the tuple, followed by the [tuple](#tuple).

### Int64

Pops a string `a` from the stack, decodes it as a [signed varint](#varint), and pushes the result to the data stack as an Int64. Fails execution if `a` is not a valid varint encoding of an integer, or if the decoded `a` is greater than or equal to `2^63`.

### Small integers

[Descriptions of opcodes that push the numbers 0-32 to the stack.]

### Pushdata

[TBD: use Keith's method for this]

## Examples

### Normal transaction

TODO: fix now that Value, Anchor, and Condition stacks are merged

    {"anchor", "anchorvalue1..."} {{"value", 5, "assetid1..."}} 1 [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey1..." checksig verify] defer] command
    {"anchor", "anchorvalue2..."} {{"value", 10, "assetid1..."}} 1 [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey2..." checksig verify] defer] command
    {"anchor", "anchorvalue3..."} {{"value", 15, "assetid2..."}} 1 [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey3..." checksig verify] defer] command
    {"anchor", "anchorvalue4..."} {{"value", 20, "assetid2..."}} 1 [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" 13 peek encode cat sha3 "pubkey4..." checksig verify] defer] command
    merge
    2 valuestack roll
    2 valuestack roll
    merge
    6 split
    [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey5..." checksig verify] defer] lock
    [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey6..." checksig verify] defer] lock
    18 split
    [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey7..." checksig verify] defer] lock
    [jumpif:$unlock lock jump:$end $unlock unlock ["txvm" txstack peek encode cat sha3 "pubkey8..." checksig verify] defer] lock
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
      ["txvm" txstack inspect encode cat sha3 "pubkey..." checksig verify] defer
    ] dup lock lock


### Issuance program signing transaction:

    [issue ["txvm" txstack inspect encode cat sha3 "pubkey..." checksig verify] defer]

Usage (to issue 5 units):

    5 [issue ["txvm" txstack inspect encode cat sha3 "pubkey..." checksig verify] defer] command


### Issuance program signing anchor:

    [0 anchorstack peek]


### Maximally flexible issuance program

    [nonce amount [issue ] command

Maximally flexible issuance program:

    [nonce]
