# TxVM

This is the specification for txvm, which combines a representation for blockchain transactions with the rules for ensuring their validity.

## Motivation

Earlier versions of Chain Core represented transactions with a static data structure, exposing the pieces of information needed to test the transaction’s validity. A separate set of validation rules could be applied to that information to get a true/false result.

Under txvm, these functions are combined in such a way that an executable program string is both the transaction’s representation and the proof of its validity.

When the virtual machine executes a txvm program, it accumulates different types of data on different stacks. This data corresponds to the information exposed in earlier versions of the transaction data structure: inputs, outputs, time constraints, nonces, and so on. Under txvm, that information is _only_ available as a result of executing the program, and the program only completes without error if the transaction is well-formed (i.e., its inputs and outputs balance, prevout control programs are correctly satisfied, etc). No separate validation steps are required.

The pieces of transaction information - the inputs, outputs, etc. - that are produced during txvm execution are also _consumed_ in order to produce the transaction ID, which is the sole output of a successful txvm program. To capture pieces of transaction information for purposes other than validation, txvm implementations can and should provide callback hooks for inspecting and copying data from the various stacks at key points during execution.

## Overview of Confidential Assets in TxVM

All values are represented as commitments by default. Perfect binding is ensured by a combination of asset ID commitment (AC, 2 points) and a value commitment (VC, 2 points).

Non-blinded commitments are done using zero blinding factors with "non-confidential range proofs":

* Non-confidential ARP contains simply an asset ID. Verifier converts asset ID to asset point A, which is then wrapped in a commitment `AC=(A,O)`.
* Non-confidential VRP contains simply an amount. Verifier multiplies amount by AC: `VC = amount*AC = (amount*A, O)`.

TxVM has 4 stacks for managing value flow:

* IC-stack: issuance candidates
* PAC-stack: Proven asset commitments
* PVC-stack: Proven value commitments
* UVC-stack: Unproven value commitments

When an input is unlocked, its AC and VC are pushed to the PAC- and PVC-stacks respectively.

When issuance is performed, the AC and VC are proven using IARP and VRP and placed to the PAC- and PVC-stacks respectively.

Note: issuance candiates, and issued AC and VC should be prepared up-front (with deferred predicates on the condition stack) so that IARP and issuance programs can sign the entire tx or introspect these values. 

**Merge** of proven value commitments does not require a proof.

**Split** takes a proven value commitment from PVC-stack (VC0), a VC from data stack (VC1), a VRP for VC1 and outputs VC2 to UVC.

**ProveAssetRange** takes an AC’ from data stack, a ARP that references ACs on PAC-stack. If ARP is valid in respect to prove ACs, the AC' is pushed to PAC-stack.

**ProveValueRange** takes VC from UVC-stack, AC from PAC-stack, VRP, verifies VRP and pushes VC unmodified to PVC-stack. VRP can be non-confidential.

**ProveAsset** pops AC, asset ID and signature from the data stack. If signature is an empty string, treats blinding factor as zero (useful for public contracts), otherwise verifies the signature as NIZKP for blinding factor (useful for in-HSM contracts). Pushes asset ID to the data stack.

**ProveValue** pops VC, AC, amount and signature from the data stack. If signature is an empty string, treats blinding factor as zero (useful for public contracts), otherwise verifies the signature as NIZKP for blinding factor (useful for in-HSM contracts). Pushes amount to the data stack.

**Issue** pops from data stack:

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

When tx is summarized, no unproven VCs must be left on the UVC-stack.


# VM Execution

The VM is initialized with all stacks empty.

When the program counter is equal to the length of the program, execution is complete. The [Transaction ID stack](#transaction-id-stack) must have one item on it. Other than the Transaction ID stack, the anchor stack, the data stack, and the alt stack, all of the stacks must be empty.

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

#### Value Commitment

0. `type`, a string, "valuecommitment"
1. `amountcommitment`, a [point](#point)
2. `blindingcommitment`, a [point](#point)

#### Asset Commitment

0. `type`, a string, "assetcommitment"
1. `assetcommitment`, a [point](#point)
2. `blindingcommitment`, a [point](#point)

#### Raw Value

0. `type`, a string, "rawvalue"
1. `valuecommitment`, a [value commitment](#value-commitment)

#### Unproven Value

0. `type`, a string, "unprovenvalue"
1. `valuecommitment`, a [value commitment](#value-commitment)
2. `assetcommitment`, an [asset commitment](#asset-commitment)

#### Proven Value

0. `type`, a string, "provenvalue"
1. `valuecommitment`, a [value commitment](#value-commitment)
2. `assetcommitment`, an [asset commitment](#asset-commitment)

#### Value

0. `type`, a string, "value"
1. `amount`, an int64
2. `assetID`, a string

#### Contract

0. `type`, a string, "contract"
1. `values`, a tuple of [value commitments](#value-commitment)
2. `program`, a [Program](#program)
3. `anchor`, a string

#### Program

0. `type`, a string, "program"
1. `script`, a string

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

#### Maxtime

0. `type`, a string, "beforeconstraint"
1. `maxtime`, an int64

#### Mintime

0. `type`, a string, "afterconstraint"
1. `mintime`, an int64

### Annotation stack

0. `type`, a string, "annotation"
1. `data`, a string

#### Transaction Summary

0. `type`, an int64
1. `inputs`, a tuple of [contracts](#contract)
2. `outputs`, a tuple of [contracts](#contract)
3. `retirements`, a tuple of [retirements](#retirement)
4. `nonces`, a tuple of [nonces](#nonces)
5. `annotations`, a tuple of [annotations](#annotation)

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

0. Data stack
1. Alt stack
2. Input stack
3. Value stack
4. Output stack
5. Condition stack
6. Nonce stack
7. Anchor stack
8. Retirement stack
9. Time Constraint stack
10. Annotation stack
11. Asset commitment stack
12. Transaction summary stack

## Data stack

Items on the data stack can be int64s, strings, or tuples.

### Alt stack

Items on the alt stack have the same types as items on the data stack. The alt stack starts out empty. Items can be moved from the data stack to the alt stack with the [toaltstack](#toaltstack) instruction, and from the alt stack to the data stack with the [fromaltstack](#fromaltstack).

### Input stack

Items on the Input stack are [Contracts](#contract) or [Legacy Outputs](#legacy-output).

### Value stack

Items on the Value stack are [Values](#value).

### Output stack

Items on the Output stack are [Contracts](#contract).

### Nonce stack

Items on the Nonce stack are [Nonces](#nonce).

### Anchor stack

Items on the anchor stack are [Anchors](#anchor).

### Condition stack

Items on the Condition stack are [Programs](#program).

### Time Constraint stack

Items on the Time Constraint stack are [Mintimes](#mintime) or [Maxtimes](#maxtime).

### Transaction Summary stack

Items on the Transaction Summary stack are [Transaction Summaries](#transaction-summary).

### Transaction ID stack

Items on the Transaction ID stack are 32-byte strings.

# Encoding formats

## Varint

TODO: Describe rules for encoding and decoding unsigned varints.

## Point

TODO: Describe rules for Ed25519 curve point encoding (including checks that should be done when decoding.)

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

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops another integer `n` from the data stack. On the stack identified by `stackid`, moves the `n`th item from the top from its current position to the top of the stack.

Fails if `stackid` does not correspond to a valid stack, or if the stack has fewer than `n + 1` items.

### Bury

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops a number `n` from the data stack. On the stack identified by `stackid`, moves the top item and inserts it at the `n`th-from-top position.

### Reverse

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops a number `n` from the data stack. On the stack identified by `stackid`, pops the top `n` items and inserts them back to the same stack in reverse order.

### Depth

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Counts the number of items on the stack identified by `stackid`, and pushes it to the data stack.

### Inspect

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Looks at the item on top of the stack identified by `stackid`, and pushes a copy of it to the data stack.

## Data stack operations

### Equal

Pops two items `val1` and `val2` from the data stack. If they have different types, or if either is a tuple, fails execution. If they have the same type: if they are equal, pushes `true` to the stack; otherwise, pushes `false` to the stack.

### Type

Looks at the top item on the data stack. Pushes a number to the stack corresponding to that item's [type](#type).

### Len

Pops a string or tuple `val` from the data stack. If `val` is a tuple, pushes the number of fields in that tuple to the data stack. If `val` is a string, pushes the length of that string to the data stack. Fails if `val` is a number.

### Drop

Drops an item from the data stack.

### Dup
	
Pops the top item from the data stack, and pushes two copies of that item to the data stack.

### ToAlt

Pops an item from the data stack and pushes it to the alt stack.

### FromAlt

Pops an item from the alt stack and pushes it to the data stack.

## Tuple operations 

### MakeTuple

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

## Entry operations

### Annotate

Pops a string, `data`, from the data stack. Pushes an [annotation](#annotation) with `data` of `data` to the Annotation stack.

### Defer

Pops a [Program](#program) from the data stack and pushes it to the Condition stack.

### Satisfy

Pops a condition from the Condition stack and executes it.

### Unlock 

Pops a tuple `input` of type [Contract](#contract) from the data stack. Pushes it to the Input stack.

Pushes each of the `values` in `input` to the Value stack, and pushes an [anchor](#anchor) to the Anchor stack with `value` equal to `input.anchor`. 

Executes `input.program`.

### UnlockOutput

Pops a [Contract](#contract) `output` from the Output stack. Pushes each of the `values` to the Value stack. Executes `output.program`.

### Merge

Pops two [Values](#value) from the Value stack. If their asset IDs are different, execution fails. Pushes a new [Value](#value) to the Value stack, whose asset ID is the same as the popped values, and whose amount is the sum of the amounts of each of the popped values.

### Split

Pops a [Value](#value) `value` from the Value stack. Pops an int64 `newamount` from the data stack. If `newamount` is greater than or equal to `value.amount`, fail execution. Pushes a new Value with amount `newamount` and assetID `value.assetID`, then pushes a new Value with amount `newamount - value.amount` and assetID `value.assetID`.

### Lock

Pops a number `n` from the data stack. Pops `n` [values](#value), `values`, from the Value stack. Pops a [Program](#program) `program` from the data stack. Pops an [anchor](#anchor) from the Anchor stack. Pushes a [Contract](#contract) to the Output stack with a tuple of the `values` as the `values`, `program` as the `program`, and the ID of `anchor` as the `anchor`.

### Retire

Pops a [Value](#value) `value` from the Value stack. Pushes a [Retirement](#retirement) to the Retirement stack. (TBD: only proven values!)

### Nonce

Pops a [nonce](#nonce) tuple `nonce` from the data stack. Verify that `nonce.blockchainid` is equal to the blockchain ID. Pushes `nonce` to the Nonce stack. Pushes an [anchor](#anchor) to the Anchor stack, with `value` equal to the [ID](#item-ids) of `nonce` . Pushes a [Maxtime](#maxtime) to the [Time Constraints stack] with `maxtime` equal to `nonce.maxtime`. Pushes a [Mintime](#mintime) to the [Time Constraints stack] with `mintime` equal to `nonce.mintime`. Executes `nonce.program`.

### Reanchor

Pops an [anchor](#anchor) `anchor` from the Anchor stack. Pushes a new anchor, with `value` set to the [ID](#item-ids) of `anchor`.

### Issue

Pops an [asset definition](#asset-definition) tuple `assetdefinition` from the data stack, and pops an int64, `amount`, from the data stack. Computes the [ID](#item-ids) `assetid` of `assetdefinition`. Pushes a [value](#value) with amount `amount` and assetID `assetID`. Executes `assetdefinition.issuanceprogram`. (TBD: confidential issuance etc).

### Before

Pops an int64 `max` from the stack. Pushes a [Maxtime](#maxtime) to the [Time Constraint stack](#time-constraint-stack) with `maxtime` equal to `max`.

### After

Pops an int64 `min` from the stack. Pushes a [Mintime](#mintime) to the [Time Constraint stack](#time-constraint-stack) with `mintime` equal to `min`.

### Summarize

Fails if the [Transaction Summary stack](#transaction-summary-stack) is not empty.

Pops all items from the Input stack and creates a tuple of them (with the top item first), `inputs`. Pops all items from the Output stack and creates a tuple of them, `outputs`. Pops all items from the Nonce stack and creates a tuple of them, `nonces`. Pops all items from the Retirement stack and creates a tuple of them, `retirements`. Pops all items from the Time Constraint stack and creates a tuple of them, `timeconstraints`. Pops all items from the Annotation stack and creates a tuple of them, `annotations`.

Creates a [transaction summary](#transaction-summary) `summary` with `inputs`, `outputs`, `nonces`, `retirements`, `timeconstraints`, and `annotations`, and pushes it to the Transaction Summary stack.

### Migrate

Pops a tuple of type [legacy output](#legacy-output) `legacy` from the data stack. Pushes it to the `inputs` stack. Pushes an [anchor](#anchor) to the Anchor stack with `value` set to the old-style ID (TBD) of `legacy`.

[TBD: parse and translate the old-style program `legacy.program`, which must be a specific format, into a new one `newprogram`.]

Pushes a [Value](#value) with amount `legacy.amount` and asset ID `legacy.assetID` to the Value stack.

Executes `newprogram`.

### Extend

Fails if the extension flag is not set. (TBD: clarify)

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Pops an item, `extension`, from the data stack. On the stack identified by `stackid`, pops a tuple and pushes a copy of that tuple with one additional field added, containing `extension`.

### IssueCA

TBD

### ProveRange

TBD

### ProveValue

TBD

### ProveAsset

TBD

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
