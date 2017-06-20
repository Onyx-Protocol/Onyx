# Stacks

## Types

There are three types of items on the VM stacks, with the following identifiers.

0. Int64
1. String
2. Tuple

"Boolean" is not a separate type, but rather the two Int64 values `0` (for false) and `1` (for true). Operations that expect booleans fail if the value on top of the stack is not `0` or `1`.

### Int64

An integer between 2^-63 and 2^63 - 1.

### String

A bytestring with length between 0 and 2^31 - 1 bytes.

### Tuple

An immutable collection of items of any type.

There are several named types of tuples.

#### Value

0. `type`, an int64
1. `history`, a 32-byte string
2. `referencedata`, a string
2. `amount`, an int64
3. `assetID`, a string

#### Output

0. `type`, an int64
1. `history`, a 32-byte string
2. `referencedata`, a string
3. `values`, a tuple of `Value`s
4. `history`, a tuple (TBD)

#### Nonce

0. `type`, an int64
1. `referencedata`, a string
2. `program`, a string
3. `mintime`, an int64
4. `maxtime`, an int64

#### Retirement

0. `type`, an int64
1. `history`, a string
2. `referencedata`, a string

#### Anchor

0. `type`, an int64
1. `history`, a string
2. `referencedata`, a string

#### Asset Definition

0. `type`, an int64
1. `history`, a string
2. `referencedata`, a string
3. `initialblockID`, a string
4. `issuanceprogram`, a string

#### Transaction Header

0. `type`, an int64
1. `referencedata`, a string
2. `outputs`, a tuple of [output IDs](#output)
3. `retirements`, a tuple of [retirement IDs](#retirement)
4. `mintime`, an int64
5. `maxtime`, an int64

## Item IDs

TBD

## Stack identifiers

0. Data stack
1. Alt stack
2. Inputs stack
3. Values stack
4. Outputs stack
5. Conditions stack
6. Nonces stack
7. Anchors stack
8. Retirements stack
9. Legacy outputs stack
10. Transaction header stack


## Data stack

Items on the data stack can be int64s, strings, or tuples.

### Alt stack

Items on the alt stack have the same types as items on the data stack. The alt stack starts out empty. Items can be moved from the data stack to the alt stack with the [toaltstack](#toaltstack) instruction, and from the alt stack to the data stack with the [fromaltstack](#fromaltstack).

### Inputs stack

Items on the inputs stack are 32-byte strings, representing IDs of [outputs](#output). The inputs stack is initialized with the IDs in the "inputs" field of the transaction header.

### Values stack

Items on the values stack are [Values](#value).

### Outputs stack

Items on the outputs stack are [Outputs](#output).

### Nonces stack

Items on the nonces stack are 32-byte strings, representing IDs of [nonces](#nonce).

### Anchors stack

Items on the anchors stack are [Anchors](#anchor).

### Conditions stack

Items on the conditions stack are strings, representing programs.

At the end of VM execution, the conditions stack must be empty.

# Encoding formats

## Varint

TODO: Describe rules for encoding and decoding unsigned varints.

## Ed25519 Curve Points

TODO: Describe rules for Ed25519 curve point encoding (including checks that should be done when decoding.)

# Operations

## Control flow operations

# Fail

Halts VM execution, returning false.

# PC

Pushes the current program counter (after incrementing for this instruction) to the data stack.

# JumpIf

Pops an integer `destination`, then a boolean `cond` from the data stack. If `cond` is false, do nothing. If `cond` is true, set program counter to `destination`. Fail if `destination` is negative, if `destination` is greater than or equal to the length of the current program.

## Stack operations 

# Roll

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops another integer `n` from the data stack. On the stack identified by `stackid`, moves the `n`th item from the top from its current position to the top of the stack.

Fails if `stackid` does not correspond to a valid stack, or if the stack has fewer than `n + 1` items.

# Bury

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks), and pops a number `n` from the data stack. On the stack identified by `stackid`, moves the top item and inserts it at the `n`th-from-top position.

# Depth

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stacks). Counts the number of items on the stack identified by `stackid`, and pushes it to the data stack.

## Data stack operations

### Equal

Pops two items `val1` and `val2` from the data stack. If they have different types, or if either is a tuple, fails execution. If they have the same type: if they are equal, pushes `true` to the stack; otherwise, pushes `false` to the stack.

### Type

Looks at the top item on the data stack. Pushes a number to the stack corresponding to that item's [type](#type).

### Encode

Pops a string or integer `val` from the data stack. Pushes a string to the data stack that, when executed on the VM, would push `val` to the data stack. Fails if `val` is a tuple.

### Len

Pops a string or tuple `val` from the data stack. If `val` is a tuple, pushes the number of fields in that tuple to the data stack. If `val` is a string, pushes the length of that string to the data stack. Fails if `val` is a number.

# Drop

Drops an item from the data stack.

# Dup
	
Pops the top item from the data stack, and pushes two copies of that item to the data stack.

# ToAlt

Pops an item from the data stack and pushes it to the alt stack.

# FromAlt

Pops an item from the alt stack and pushes it to the data stack.

## Tuple operations 

### Tuple

Pops an integer `len` from the data stack. Pops `len` items from the data stack and creates a tuple of length `len` on the data stack.

### Untuple

Pops a tuple `tuple` from the data stack. Pushes each of the fields in `tuple` to the stack in reverse order (so that the 0th item in the tuple ends up on top of the stack).

### Field

Pops an integer `stackid` from the data stack, representing a [stack identifier](#stack-identifier), and pops another integer `i` from the top of the data stack. Looks at the tuple on top of the stack identified by `stackid`, and pushes the item in its `i`th field to the top of the data stack.

Fails if the stack identified by `stackid` is empty or does not have a tuple of at least length `i + 1` on top of it, or if `i` is negative.

## Boolean operations

### Not

Pop a boolean `p` from the stack. If `p` is `true`, push `false`. If `p` is `false`, push `true`.

### And

Pops two booleans `p` and `q` from the stack. If both `p` and `q` are true, push `true`. Otherwise, push `false`.

### Or

Pops two booleans `p` and `q` from the stack. If both `p` and `q` are false, push `false`. Otherwise, push `true`.

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

Pops two integers, `start`, then `end`, from the stack. Pops a string `str` from the stack. Pushes the string `str[start:end]` (with the first character being the one at index `start`, and the second character being the one at index `end`). Fails if `end` is less than `start`, if `start` is less than 0, or if `end` is greater thanss the length of `str`.

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

TODO: add descriptions.

### Defer

Pops a string from the data stack and pushes it as an condition to the conditions stack.

### Satisfy

Pops a condition from the conditions stack and executes it.

### Unlock

Pops a string `inputid` from the Inputs stack. Pops a tuple of type [Output](#output) from the data stack. Verifies that the [id](#Item-ID) of the tuple matches `inputid`. Pushes its `program` as a string to the Conditions stack, and pushes each of the `values` to the Values stack.

### UnlockOutput

Pops an output `output` from the Outputs stack. Pushes its `program` as a string to the Conditions stack, and pushes each of the `values` to the Values stack.

### Merge

Pops two [Values](#value) from the Values stack. If their asset IDs are different, execution fails. Pushes a new [Value](#value) to the Values stack, whose asset ID is the same as the popped values, and whose amount is the sum of the amounts of each of the popped values.

### Split

Pops a [Value](#value) `value` from the Values stack. Pops an int64 `newamount` from the data stack. If `newamount` is greater than or equal to `value.amount`, fail execution. Pushes a new Value with amount `newamount` and assetID `value.assetID`, then pushes a new Value with amount `newamount - value.amount` and assetID `value.assetID`.

### Lock

Pops a string `referencedata` from the data stack. Pops a number `n` from the data stack. Pops `n` [values](#value), `values`, from the Values stack. Pops a string `program` from the data stack. Pushes an [output](#output) to the Outputs stack with `referencedata` as the `referencedata`, a tuple of the `values` as the `values`, and `program` as the `program`.

### Retire

Pops a [Value](#value) `value` from the Values stack. Pushes a retirement to the retirements stack.

### Anchor

Pop a [nonce](#nonce) tuple from the data stack. Pop a string `nonceID` from the nonces stack. Verify that the ID of the `nonce` is equal to `nonceID`. Push an [anchor](#anchor) to the anchors stack, and push `nonce.program` as a condition to the conditions stack.

### Issue

Pop an [asset definition](#asset-definition) tuple `assetdefinition` from the data stack, and pops an int64, `amount`, from the data stack. Push `assetdefinition.issuanceprogram` as a condition to the conditions stack. Compute an assetID `assetID` from `assetdefinition`. Push a [value](#value) with amount `amount` and assetID `assetID`.

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

Have no effect when executed.

### Reserved

Causes the VM to halt and fail.

## Encoding opcodes

### Int64

Pops a string `a` from the stack, decodes it as a [varint](#varint), and pushes the result to the data stack as an Int64. Fails execution if `a` is not a valid varint encoding of an integer, or if the decoded `a` is greater than or equal to `2^63`.

### Negate

Pops an integer `x` from the data stack, negates it, and pushes the result `-x` to the data stack.

### Small integers

[Descriptions of opcodes that push the numbers 0-32 to the stack.]

### Pushdata

Followed by an integer `n` encoded as a varint, then `n` bytes of data. Fails if `n` is greater than 2^31.