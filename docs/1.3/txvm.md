# Stacks

0. Data stack
1. Alt stack
2. Inputs stack
3. Values stack
4. Outputs stack
5. Conditions stack
6. Nonces stack
7. Anchors stack
8. Transaction header stack
9. Muxes stack

# Typess

0. Int64
1. String
2. Tuple

# Operations

## Control flow operations

# Fail

Halts VM execution, returning false.

# PC

Pushes the current program counter to the data stack.

# JumpIf

Pops a number `destination`, then a boolean `cond` from the data stack. If `condition` is false, do nothing. If `cond` is true, set program counter to `destination`.

## Stack operations 

# Roll

Pops a number `stackid` from the data stack, representing a [stack identifier](#stacks), and pops another number `n` from the data stack. On the stack identified by `stackid`, moves the `n`th item from the top from its current position to the top of the stack.

Fails if `stackid` does not correspond to a valid stack, or if the stack has fewer than `n + 1` items.

# Bury

Pops a number `stackid` from the data stack, representing a [stack identifier](#stacks), and pops a number `n` from the data stack. On the stack identified by `stackid`, moves the top item and inserts it at the `n`th-from-top position.

# Depth

Pops a number `stackid` from the data stack, representing a [stack identifier](#stacks). Counts the number of items on the stack identified by `stackid`, and pushes it to the data stack.

## Data stack operations

### Equal

Pops two items `val1` and `val2` from the data stack. If they have different types, or if either is a tuple, fails execution. If they have the same type, .

### Type

Looks at the top item on the data stack. Pushes a number to the stack corresponding to that item's [type](#type).

### Encode

Pops a string or number `val` from the data stack. Pushes a string to the data stack that, when executed on the VM, would push `val` to the data stack. Fails if `val` is a tuple.

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

Pops a number `len` from the data stack. Pops `len` items from the data stack and creates a tuple of length `len` on the data stack.

### Untuple

Pops a tuple `tuple` from the data stack. Pushes each of the fields in `tuple` to the stack in reverse order (so that the 0th item in the tuple ends up on top of the stack).

### Field

Pops a number `stackid` from the data stack, representing a [stack identifier](#stack-identifier), and pops another number `i` from the top of the data stack. Looks at the tuple on top of the stack identified by `stackid`, and pushes the item in its `i`th field to the top of the data stack.

Fails if the stack identified by `stackid` is empty or does not have a tuple of at least length `i + 1` on top of it.

## Boolean operations

	Not   = 19
	And   = 20
	Or    = 21
	GT    = 22
	GE    = 23

### Add
	
### Sub

### Mul

### Div

### Mod

## String operations

### Cat

### Slice

## Bitwise operations

[These work on either two Int64s, or two strings. If the strings have different lengths, I think the operation should fail.]

### BitNot

### BitAnd

### BitOr

### BitXor

## Crypto operations

### SHA256

### SHA3

### CheckSig

### PointAdd

### PointSub

### PointMul

## Entry operations

TODO: add descriptions.

	Cond         = 46 // prog => cond
	Unlock       = 47 // inputid + data => value + cond
	UnlockOutput = 48 // outputid + data => value + cond
	Merge        = 49 // value value => value
	Split        = 50 // value + amount => value value
	ProveRange   = 51 // TODO(kr): review for CA
	ProveValue   = 52 // TODO(kr): review for CA
	ProveAsset   = 53 // TODO(kr): review for CA
	Blind        = 54 // TODO(kr): review for CA
	Lock         = 55 // value + prog => outputid
	Satisfy      = 56 // cond => {}
	Anchor       = 57 // nonce + data => anchor + cond
	Issue        = 58 // anchor + data => value + cond
	IssueCA      = 59 // TODO(kr): review for CA
	Retire       = 60 // valud + refdata => {}

## Legacy operations

	VM1CheckPredicate = 63 // list vm1prog => bool
	VM1Unlock         = 64 // vm1inputid + data => vm1value + cond
	VM1Nonce          = 65 // vm1nonce => vm1anchor + cond
	VM1Issue          = 66 // vm1anchor => vm1value + cond
	VM1Mux            = 67 // entire vm1value stack => vm1mux
	VM1Withdraw       = 68 // vm1mux + amount asset => vm1mux + value

## Extension opcodes

	Nop0    = 69
	Nop1    = 70
	Nop2    = 71
	Nop3    = 72
	Nop4    = 73
	Nop5    = 74
	Nop6    = 75
	Nop7    = 76
	Nop8    = 77
	Private = 78

## Encoding opcodes

### Negate

### Small integers

### Pushdata

