# Virtual Machine Specification

* [Introduction](#introduction)
* [Versioning](#versioning)
* [Program format](#program-format)
* [Execution context](#execution-context)
* [VM state](#vm-state)
* [Operations](#operations)
* [Instruction cost](#instruction-cost)
* [Value types](#value-types)
* [Failure conditions](#failure-conditions)
* [Instructions](#instructions)
  * [Instructions pushing data on stack](#instructions-pushing-data-on-stack)
  * [Control flow operators](#control-flow-operators)
  * [Stack control operators](#stack-control-operators)
  * [Splice operators](#splice-operators)
  * [Bitwise operators](#bitwise-operators)
  * [Logical and numeric operators](#logical-and-numeric-operators)
  * [Cryptographic instructions](#cryptographic-instructions)
  * [Introspection instructions](#introspection-instructions)
  * [Expansion opcodes](#expansion-opcodes)
* [References](#references)


## Introduction

The Chain Protocol uses a bytecode language to express short programs used to authenticate blockchain activity. Programs are used in [two contexts](#execution-context): transactions and blocks.

Programs are executed in a stack-based [virtual machine](#vm-state).

* First, the program arguments are pushed on the stack one after the other (so that the last argument is on the top of the stack).
* Then the VM executes the actual predicate program (control program, issuance program or consensus program) encoded as a sequence of **opcodes**.
* If execution halts early (because of a disabled opcode, [FAIL](#fail), a [VERIFY](#verify) failure, or exceeding the run limit), validation fails.
* If execution completes successfully, the top stack value is inspected. If it’s zero, validation fails, otherwise validation succeeds.

Each instruction has a built-in [run cost](#instruction-cost) that counts against a built-in *run limit* to protect the network from resource exhaustion. Currently, the protocol [mandates](#vm-state) a specific run limit. Future VM versions will provide more fine-grained control over run limit by operators and users of the network.

## Versioning

Virtual machines for control and issuance programs inside transactions are versioned to enable future improvements using varint-encoded version number in the commitment strings where these programs are specified. Program arguments are not explicitly versioned, their semantics are defined by the VM version of the associated issuance and control program.

Nodes ignore programs with unknown versions, treating them like “anyone can issue/spend.” To discourage use of unassigned versions, block signers refuse to include transactions that use unassigned VM versions.

Blocks do not specify VM version explicitly. [Consensus programs](blockchain.md#output-1) use VM version 1 with additional [block-context restrictions](#block-context) applied to some instructions. Upgrades to block authentication can be made via additional fields in the block header.


## Program format

A program comprises a sequence of zero or more **instructions**. Each instruction contains a one-byte **opcode** followed by zero or more **continuation bytes**, determined by the operation. Data in this format is informally known as **bytecode**.

Instructions that push arbitrary data onto the stack use one of the [PUSHDATA](#pushdata) opcode followed by a variable-length binary string to be placed on stack. The length of the string is either encoded within the opcode itself, or prepended to the string.

All other instructions are encoded simply by a single-byte opcode. The protocol reserves unassigned opcodes for future extensions.


## Execution context

A program executes in a context, either a *block* or a *transaction*. Some instructions have different meaning based on the context.

Transactions use [control programs](blockchain.md#output-1) to define predicates governing spending of an asset in a later transaction, *issuance programs* for predicates authenticating issuance of an asset, and *program arguments* to provide input data for the predicates in output and issuance programs.

Blocks use [consensus programs](blockchain.md#block-header) to define predicates for signing the next block and *program arguments* to provide input data for the predicate in the previous block. Consensus programs have restricted functionality and do not use version tags. Some instructions (such as [ASSET](#asset) or [CHECKOUTPUT](#checkoutput)) that do not make sense within the context of signing a block are disabled and cause an immediate validation failure.

### Block context

Block context is defined by the block necessary for [BLOCKHASH](#blockhash), [NEXTPROGRAM](#nextprogram) and [BLOCKTIME](#blocktime) execution.

Instruction [PROGRAM](#program) behaves differently than in transaction context.

Execution of any of the following instructions results in immediate failure:

* [TXSIGHASH](#txsighash)
* [CHECKOUTPUT](#checkoutput)
* [ASSET](#asset)
* [AMOUNT](#amount)
* [MINTIME](#mintime)
* [MAXTIME](#maxtime)
* [TXDATA](#txdata)
* [ENTRYDATA](#entrydata)
* [INDEX](#index)
* [OUTPUTID](#outputid)
* [NONCE](#nonce)


### Transaction context

Transaction context is defined by the pair of the entire transaction and a “current entry” that contains the evaluated predicate (e.g. a [spend](blockchain.md#spend-1), [issuance](blockchain.md#issuance-1) or [mux](blockchain.md#mux-1)).

Execution of any of the following instructions results in immediate failure:

* [BLOCKHASH](#blockhash)
* [NEXTPROGRAM](#nextprogram)
* [BLOCKTIME](#blocktime)


## VM state

**VM State** is a tuple consisting of:

1. Program
2. Program Counter (PC)
3. Data Stack
4. Alt Stack
5. Run Limit
6. Expansion Flag
7. Execution Context:
    a. Block
    b. (Transaction, Current Entry)

**Initial State** has empty stacks, uninitialized program, PC set to zero, and *run limit* set to 10,000.

**Program** is a sequence of instructions, encoded as bytecode.

**PC** is a 32-bit unsigned integer used as a pointer to an opcode within a program.

**Data Stack** and **Alt Stack** are stacks of binary strings.

**Run Limit** is a built-in 64-bit integer specifying remaining total cost of execution. Run limit is decreased by the cost of each instruction and also affected by data added to and removed from the data stack and alt stack. Every byte added to either stack costs 1 unit, and every byte removed from either stack refunds 1 unit. (This includes explicit additions and removals by stack-manipulating instructions such as PUSHDATA and DROP, and also implicit additions and removals as when other instructions consume arguments and produce results.)

**Expansion Flag** indicates whether the [expansion opcodes](#expansion-opcodes) are allowed in the program or not. If the flag is off, these opcodes immediately fail the program execution.

**Execution Context** is either a [block context](#block-context) or [transaction context](#transaction-context).


## Operations

The VM has two high-level operations: [Prepare VM](#prepare-vm) and [Verify Predicate](#verify-predicate).

### Prepare VM

Places program arguments on the data stack one after another so that last argument is on the top of the stack. Additions to the stack are counted against the run limit using the [standard memory cost](#standard-memory-cost) function.

### Verify predicate

Initializes VM with a predicate program and begins its execution with PC set to zero.

At the beginning of each execution step, the PC is checked. If it is less than the length of the program, VM reads the opcode at that byte position in the program and executes a corresponding instruction. Instructions are executed as described in the [Instructions](#instructions) section. The run limit is decreased or increased according to the instruction’s *run cost*. If the instruction’s run cost exceeds the current run limit, the instruction is not executed and execution fails immediately.

If the PC is equal to or greater than the length of the program at the beginning of an execution step, execution is complete, and the top value of the data stack is checked and interpreted as a boolean. If it is `false`, or if the data stack is empty, verification fails; otherwise, verification succeeds. (Note: The data stack may contain any number of elements when execution finishes; there is no "clean stack" requirement. The alt stack also can be non-empty upon completion.)

After each step, the PC is incremented by the size of the current instruction.


## Instruction cost

Every instruction has a cost that affects VM *run limit*. Total instruction cost consists of *execution cost* and *memory cost*. Execution cost always reduces remaining run limit, while memory usage cost can be refunded (increasing the run limit) when previously used memory is released during VM execution.

### Execution cost

Every instruction has a constant or variable execution cost. Simple instructions such as [ADD](#add) have constant execution cost. More complex instructions like [SHA3](blockchain.md#sha3) or [CHECKSIG](#checksig) have cost depending on amount of data that must be processed.

In order to account for spikes in memory usage some instructions (e.g. [CAT](#cat)) define a cost and a refund: before execution begins the cost is applied to the run limit, then after completion refund is applied together with run limit changes due to memory usage.

### Memory cost

Memory cost is incurred when additional memory is allocated. This cost is fully refundable when the memory is released. Most operations allocate and release memory by using the data stack, but some others also use the alt stack ([TOALTSTACK](#toaltstack), [FROMALTSTACK](#fromaltstack)) and the system memory for new VM instances ([CHECKPREDICATE](#checkpredicate)).

### Standard memory cost

Most instructions use only the data stack by removing some items and then placing some items back on the stack. For these operations, we define the *standard memory cost* applied as follows:

1. Instruction’s memory cost value is set to zero.
2. For each item removed from the data stack, instruction’s memory cost is decreased by 8+L where L is the length of the item in bytes.
3. For each item added to the data stack the cost is increased by 8+L where L is the length of the item in bytes.


## Value types

All items on the data and alt stacks are binary strings. Some instructions accept or return items of other types. When values of those types are pushed to or popped from the data stack, they are coerced to and from strings in accordance with the rules specified below.

In the stack diagrams accompanying the definition of each operation, `x` and `y` denote [numbers](#vm-number), `m` and `n` denote non-negative [numbers](#vm-number), and `p` and `q` denote [booleans](#vm-boolean). If coercion fails (or if stack items designated as `m` or `n` coerce to negative numbers), the operation fails.

### VM String

An ordered sequence of 0 or more bytes.

In this document, single bytes are represented in hexadecimal form with a `0x` base prefix, i.e. `0x00` or `0xff`.

In this document, strings are represented as sequences of unprefixed hexadecimal bytes, separated by spaces, and enclosed in double quotation marks, i.e. `""`, `"01"`, or `"ff ff"`.

### VM Boolean

A boolean value (`true` or `false`).

#### String to Boolean

Any string can be coerced to a boolean.

Strings coerce to `true` if and only if they contain any non-zero bytes. Therefore, for example, `""`, `"00"`, and `"00 00"` coerce to `false`.

#### Boolean to String

`false` coerces to an empty string `""` (the same representation as the number 0), `true` coerces to a one-byte string `"01"` (the same representation as the number 1).

### VM Number

An integer greater than or equal to –2<sup>63</sup>, and less than 2<sup>63</sup>.

Certain arithmetic operations use conservative bounds checks (explicitly specified below) on their inputs to prevent the output from being outside the legal range. If one of these bounds checks fails, execution fails.

#### String to Number

1. If the string is longer than 8 bytes, fail execution.
2. If the string is shorter than 8 bytes, right-pad it by appending `0x00` bytes to get an 8-byte string.
3. Interpret the 8-byte string as a [little-endian](https://en.wikipedia.org/wiki/Endianness#Little-endian) 64-bit integer, using [two's complement representation](https://en.wikipedia.org/wiki/Two%27s_complement).

#### Number to String

1. Create an 8-byte string matching the representation of the number as a [little-endian](https://en.wikipedia.org/wiki/Endianness#Little-endian) 64-bit integer, using [two's complement representation](https://en.wikipedia.org/wiki/Two%27s_complement) for negative integers.
2. Trim the string by removing any `0x00` bytes from the right side.

Value          | String (hexadecimal)        | Size in bytes
---------------|-----------------------------|------------------
0              | `“”`                        | 0
1              | `“01”`                      | 1
–1             | `“ff ff ff ff ff ff ff ff”` | 8
2^63 - 1 (max) | `“ff ff ff ff ff ff ff 7f”` | 8
-2^63 (min)    | `“00 00 00 00 00 00 00 80”` | 8


## Failure conditions

Validation fails when:

* an instruction is executed that expects more elements on the stack than are present (see stack diagrams below under [Instructions](#instructions))
* a [VERIFY](#verify) instruction fails
* a [FAIL](#fail) instruction is executed
* the run limit is below the value required by the current instruction
* an invalid encoding is detected for keys or [signatures](blockchain.md#signature)
* coercion fails for [numbers](#vm-number)
* a bounds check fails for one of the [splice](#splice-operators) or [numeric](#logical-and-numeric-operators) instructions
* the program execution finishes with an empty data stack
* the program execution finishes with a [false](#vm-boolean) item on the top of the data stack
* an instruction specifies that it fails (see below)


## Instructions

This section specifies the behavior of every instruction. Each instruction has a name and a run cost that modifies the VM’s run limit.

The cost of an instruction may be fixed or variable based on the amount of data being processed. In the tables below, the notation L or L<sub>x</sub> is used to indicate the length in bytes of a given binary string. Positive cost increases reduces (“consumes”) run limit, negative cost increases (“refunds”) run limit.

When the instruction cost is specified as a single value, it is applied to the run limit *before* the instruction is executed. If two values represent a cost (such as `1; -1` or `1; standard memory cost`) that means that the first value is applied *before* executing the instruction and the second value is applied *after*.

Execution immediately halts if the run limit is insufficient to apply the cost of the instruction. In such case, the run limit is left unchanged (instead of becoming negative) and the execution halts. If the instruction defines two cost values (before and after the execution), and the first one did not cause the VM to halt, then its effects are not reversed. E.g. if the instruction cost is defined as `2;6` and the run limit is 5, then the first cost (2) is applied successfully (run limit becomes 3) and the second cost (6) halts execution leaving the run limit at 3. This value then can be refunded to the parent VM of a [CHECKPREDICATE](#checkpredicate) instruction.

Stack diagrams tell how top items of the data stack are replaced with new items. E.g. a stack diagram `(a b → c d)` says that the topmost item `b` and preceding item `a` are removed from the data stack and items `c` and `d` are pushed one after another.


### Non-executed instructions

Typically, all instructions must be executed in order for the execution to succeed. However, [JUMP](#jump) and [JUMPIF](#jumpif) instructions may cause the program to skip some of the instructions by “jumping over” them. If that happens, those instructions are not executed. This also means, that if those instructions are unassigned and [reserved for future expansion](#expansion-opcodes), they do not cause the execution to fail even if [expansion flag](#vm-state) is off.


### Instructions pushing data on stack

#### FALSE

Alias: `OP_0`.

Code  | Stack Diagram     | Cost
------|-------------------|-----------------------------------------------------
0x00  | (∅ → 0)           | 1; [standard memory cost](#standard-memory-cost)

Pushes an empty string (the [VM number](#vm-number) 0) to the data stack.


#### PUSHDATA

Code          | Stack Diagram   | Cost
--------------|-----------------|-----------------------------------------------------
0x01 to 0x4e  | (∅ → a)         | 1 + [standard memory cost](#standard-memory-cost)

Each opcode **0x00 ≤ n ≤ 0x4b** is followed by `n` bytes of data to be pushed onto the data stack as a single [VM string](#vm-string). So opcode 0x01 is followed by 1 byte of data, 0x09 by 9 bytes, and so on up to 0x4b (75 bytes).

Opcode **0x4c** is followed by a 1-byte length prefix encoding a length `n`, then `n` bytes of data to push (supports up to 255 bytes).

Opcode **0x4d** is followed by a 2-byte little-endian length prefix encoding a length `n`, then `n` bytes of data to push (supports up to 65535 bytes).

Opcode **0x4e** is followed by a 4-byte little-endian length prefix encoding a length `n`, then `n` bytes of data to push (supports up to 4294967295 bytes).

Each of these operations fails if they are not followed by the expected number of bytes of data.

#### 1NEGATE

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0x4f  | (∅ → –1)        | 1 + [standard memory cost](#standard-memory-cost)

Pushes `"ff ff ff ff ff ff ff ff"` (the [VM number](#vm-number) -1) onto the data stack.


#### OP\_1 to OP\_16

Name | Code  | Stack Diagram  | Cost                                              | Description
-----|-------|----------------|---------------------------------------------------|---------------
OP_1 | 0x51  | (∅ → 1)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 1 on the data stack.
OP_2 | 0x52  | (∅ → 2)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 2 on the data stack.
OP_3 | 0x53  | (∅ → 3)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 3 on the data stack.
OP_4 | 0x54  | (∅ → 4)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 4 on the data stack.
OP_5 | 0x55  | (∅ → 5)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 5 on the data stack.
OP_6 | 0x56  | (∅ → 6)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 6 on the data stack.
OP_7 | 0x57  | (∅ → 7)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 7 on the data stack.
OP_8 | 0x58  | (∅ → 8)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 8 on the data stack.
OP_9 | 0x59  | (∅ → 9)        | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 9 on the data stack.
OP_10 | 0x5a | (∅ → 10)       | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 10 on the data stack.
OP_11 | 0x5b | (∅ → 11)       | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 11 on the data stack.
OP_12 | 0x5c | (∅ → 12)       | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 12 on the data stack.
OP_13 | 0x5d | (∅ → 13)       | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 13 on the data stack.
OP_14 | 0x5e | (∅ → 14)       | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 14 on the data stack.
OP_15 | 0x5f | (∅ → 15)       | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 15 on the data stack.
OP_16 | 0x60 | (∅ → 16)       | 1 + [standard memory cost](#standard-memory-cost) | Pushes [number](#vm-number) 16 on the data stack.



### Control Flow Operators

#### JUMP

Code  | Stack Diagram     | Cost
------|-------------------|----------------------------
0x63  | (∅ → ∅)           | 1

Followed by a 4-byte unsigned integer `address`.

Sets the PC to `address`.

Fails if not followed by 4 bytes.

Note: this opcode may cause some instructions [to not be executed](#non-executed-instructions).


#### JUMPIF

Code  | Stack Diagram     | Cost
------|-------------------|----------------------------
0x64  | (p → ∅)           | 1; [standard memory cost](#standard-memory-cost)

Followed by a 4-byte unsigned integer `address`.

Pops a [boolean](#vm-boolean) from the data stack. If it is `true`, sets the PC to `address`. If it is `false`, does nothing.

Fails if not followed by 4 bytes.

Note: this opcode may cause some instructions [to not be executed](#non-executed-instructions).


#### VERIFY

Code  | Stack Diagram     | Cost
------|-------------------|----------------------------
0x69  | (p → ∅)           | 1; [standard memory cost](#standard-memory-cost)

Fails execution if the top item on the data stack is [false](#vm-boolean). Otherwise, removes the top item.


#### FAIL

Code  | Stack Diagram     | Cost
------|-------------------|----------------------------
0x6a  | (∅ → ∅)           | 1

Fails execution unconditionally.



#### CHECKPREDICATE

Code  | Stack Diagram            | Cost
------|--------------------------|----------------------------
0xc0  | (n predicate limit → q)  | 256 + limit; [standard memory cost](#standard-memory-cost) – 256 + 64 – leftover

If the remaining run limit is less than 256, execution fails immediately.

1. Pops 3 items from the data stack: `limit`, `predicate` and `n`.
2. Coerces `limit` to an [integer](#vm-number).
3. Coerces `n` to an [integer](#vm-number).
4. If `limit` equals zero, sets it to the VM's remaining run limit minus 256.
5. Reduces VM’s run limit by `256 + limit`.
6. Instantiates a new VM instance (“child VM”) with its run limit set to `limit`.
7. Moves the top `n` items from the parent VM’s data stack to the child VM’s data stack without incurring run limit refund or charge of their [standard memory cost](#standard-memory-cost) in either VM. The order of the moved items is unchanged. The memory cost of these items will be refunded when the child VM pops them, or when the child VM is destroyed and its parent VM is refunded.
8. Child VM evaluates the predicate and pushes `true` to the parent VM data stack if the evaluation did not fail and the child VM’s data stack is non-empty with a `true` value on top (this implements the same semantics as for the top-level [verify predicate](#verify-predicate) operation). It pushes `false` otherwise. Note that the parent VM does not fail when the child VM exhausts its run limit or otherwise fails.
9. After the child VM finishes execution (normally or due to a failure), the parent VM’s run limit is refunded with a `leftover` value computed as a sum of the following values:
    1. Remaining run limit of the child VM.
    2. [Standard memory cost](#standard-memory-cost) of all items left on the child VM’s data stack.
    3. [Standard memory cost](#standard-memory-cost) of all items left on the child VM’s alt stack.
10. The total post-execution cost is then calculated as a sum of the following values:
    1. Refund of the [standard memory cost](#standard-memory-cost) of the top three items on the parent’s data stack (`limit`, `predicate`, `n`).
    2. –256 (refunds cost of allocating memory for the child VM).
    3. +64 (cost of creating the child VM).
    4. `–leftover` (refund for the unused run limit and released memory within the child VM).

Failure conditions:

* `n` is not a non-negative [number](#vm-number), or
* there are less than `n+3` items on the data stack (including `n`, `predicate`, `limit`), or
* `limit` is not a non-negative [number](#vm-number), or
* the run limit is less than 256, or
* the run limit is less than `256+limit`.


### Stack control operators


#### TOALTSTACK

Code  | Stack Diagram      | Cost
------|--------------------|----------------------------
0x6b  | (a → ∅)            | 2

Moves the top item from the data stack to the alt stack.


#### FROMALTSTACK

Code  | Stack Diagram      | Cost
------|--------------------|----------------------------
0x6c  | (∅ → a)            | 2

Moves the top item from the alt stack to the data stack.

Fails if the alt stack is empty.


#### 2DROP

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x6d  | (a b → ∅)       | 2 + [standard memory cost](#standard-memory-cost)

Removes top 2 items from the data stack.


#### 2DUP

Code  | Stack Diagram         | Cost
------|-----------------------|----------------------------
0x6e  | (a b → a b a b)       | 2 + [standard memory cost](#standard-memory-cost)

Duplicates top 2 items on the data stack.


#### 3DUP

Code  | Stack Diagram         | Cost
------|-----------------------|----------------------------
0x6f  | (a b c → a b c a b c) | 3 + [standard memory cost](#standard-memory-cost)

Duplicates top 3 items on the data stack.


#### 2OVER

Code  | Stack Diagram           | Cost
------|-------------------------|----------------------------
0x70  | (a b c d → a b c d a b) | 2 + [standard memory cost](#standard-memory-cost)

Duplicates two items below the top two items on the data stack.


#### 2ROT

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x71  | (a b c d e f → c d e f a b) | 2

Moves 2 items below the top 4 items on the data stack to the top of the stack.


#### 2SWAP

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x72  | (a b c d → c d a b)         | 2

Moves 2 items below the top 2 items on the data stack to the top of the stack.


#### IFDUP

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x73  | (a → a \| a a)              | 1 + [standard memory cost](#standard-memory-cost)

Duplicates the top item only if it’s not [false](#vm-boolean).


#### DEPTH

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x74  | (∅ → x)                     | 1; [standard memory cost](#standard-memory-cost)

Adds the size of the data stack encoded as a [VM number](#vm-number).


#### DROP

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x75  | (a → ∅)                     | 1; [standard memory cost](#standard-memory-cost)

Removes the top item from the data stack.


#### DUP

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x76  | (a → a a)                   | 1 + [standard memory cost](#standard-memory-cost)

Duplicates the top item on the data stack.


#### NIP

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x77  | (a b → b)                   | 1 + [standard memory cost](#standard-memory-cost)

Removes the item below the top one on the data stack.


#### OVER

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x78  | (a b → a b a)               | 1 + [standard memory cost](#standard-memory-cost)

Copies the second from the top item to the top of the data stack.


#### PICK

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x79  | (a<sub>n</sub> ... a<sub>1</sub> a<sub>0</sub> n → a<sub>n</sub> ... a<sub>1</sub> a<sub>0</sub> a<sub>n</sub>)  | 2 + [standard memory cost](#standard-memory-cost)

Copies `n+2`th item from the top to the top of the data stack.

Fails if the top item is not a valid non-negative number or there are fewer than `n+2` items on the stack.


#### ROLL

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x7a  | (a<sub>n</sub> ... a<sub>1</sub> a<sub>0</sub> n → a<sub>n-1</sub> ... a<sub>1</sub> a<sub>0</sub> a<sub>n</sub>)  | 2 + [standard memory cost](#standard-memory-cost)

Moves `n+2`th item from the top to the top of the data stack.

Fails if the top item is not a valid non-negative number or there are fewer than `n+2` items on the stack.


#### ROT

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x7b  | (a b c → b c a)             | 2

Moves the third item from the top to the top of the data stack.


#### SWAP

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x7c  | (a b → b a)                 | 1

Swaps top two items on the data stack.


#### TUCK

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x7d  | (a b → b a b)               | 1 + [standard memory cost](#standard-memory-cost)

Tucks the second item from the top of the data stack with two copies of the top item.



### Splice operators


#### CAT

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0x7e  | (“a” “b” → “ab”)            | 4 + L<sub>a</sub> + L<sub>b</sub>; –(L<sub>a</sub> + L<sub>b</sub>) + [standard memory cost](#standard-memory-cost)

Concatenates top two items on the data stack.


#### SUBSTR

Code  | Stack Diagram                    | Cost
------|----------------------------------|----------------------------
0x7f  | (string m n → substring) | 4 + size; –size + [standard memory cost](#standard-memory-cost)

Extracts a substring of `string` of a given size `n` at a given offset `m`.

Failure conditions:

* `n` is not a [VM number](#vm-number), or
* `n` is negative, or
* `n` is greater than the byte size of the `string`, or
* `m` is not a [VM number](#vm-number), or
* `m` is not in range of [0, L<sub>string</sub> – size].


#### LEFT

Code  | Stack Diagram                   | Cost
------|---------------------------------|----------------------------
0x80  | (string n → prefix)          | 4 + size; –size + [standard memory cost](#standard-memory-cost)

Extracts a prefix of `string` with the given size `n`.

Failure conditions:

* `n` is not a [VM number](#vm-number), or
* `n` is negative, or
* `n` is greater than the byte size of the `string`.

#### RIGHT

Code  | Stack Diagram                   | Cost
------|---------------------------------|----------------------------
0x81  | (string n → suffix)          | 4 + size; –size + [standard memory cost](#standard-memory-cost)

Extracts a suffix of `string` with the given size `n`.

Failure conditions:

* `n` is negative, or
* `n` is greater than the byte size of the `string`.


#### SIZE

Code  | Stack Diagram                   | Cost
------|---------------------------------|----------------------------
0x82  | (string → string n)          | 1; [standard memory cost](#standard-memory-cost)

Pushes the size of `string` encoded as a [number](#vm-number) `n` without removing `string` from the data stack.


#### CATPUSHDATA

Code  | Stack Diagram                | Cost
------|------------------------------|----------------------------
0x89  | (“a” “b” → “a pushdata(b)”)  | 4 + L<sub>a</sub> + L<sub>b</sub>; –(L<sub>a</sub> + L<sub>b</sub>) + [standard memory cost](#standard-memory-cost)

Appends second string encoded as the most compact [PUSHDATA](#pushdata) instruction. This is used for building new programs piecewise.




### Bitwise operators


#### INVERT

Code  | Stack Diagram                | Cost
------|------------------------------|----------------------------
0x83  | (a → ~a)                     | 1 + L<sub>x</sub>

Inverts bits in the first item on the data stack.


#### AND

Code  | Stack Diagram                | Cost
------|------------------------------|----------------------------
0x84  | (a b → a&b)                  | 1 + min(L<sub>a</sub>,L<sub>b</sub>); [standard memory cost](#standard-memory-cost)

Bitwise AND operation. Longer item is truncated, keeping the prefix.


#### OR

Code  | Stack Diagram                | Cost
------|------------------------------|----------------------------
0x85  | (a b → a\|b)                 | 1 + max(L<sub>a</sub>,L<sub>b</sub>); [standard memory cost](#standard-memory-cost)

Bitwise OR operation. Shorter item is zero-padded to the right.


#### XOR

Code  | Stack Diagram                | Cost
------|------------------------------|----------------------------
0x86  | (a b → a^b)                  | 1 + max(L<sub>a</sub>,L<sub>b</sub>); [standard memory cost](#standard-memory-cost)

Bitwise XOR operation. Shorter item is zero-padded to the right.


#### EQUAL

Code  | Stack Diagram                | Cost
------|------------------------------|----------------------------
0x87  | (a b → a == b)                 | 1 + min(L<sub>a</sub>,L<sub>b</sub>); [standard memory cost](#standard-memory-cost)

Pops two strings from the stack and compares them byte-by-byte. Pushes [true](#vm-boolean) if the strings are equal, [false](#vm-boolean) otherwise.


#### EQUALVERIFY

Code  | Stack Diagram                | Cost
------|------------------------------|----------------------------
0x88  | (a b → ∅)                    | 1 + min(L<sub>a</sub>,L<sub>b</sub>); [standard memory cost](#standard-memory-cost)

Same as [EQUAL](#equal) [VERIFY](#verify). Pops two strings from the stack, compares them byte-by-byte, and fails execution if they are not equal.



### Logical and numeric operators


#### 1ADD

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x8b  | (x → x+1)       | 2; [standard memory cost](#standard-memory-cost)

Pops a [number](#vm-number) from the data stack, adds 1 to it, and pushes the result to the data stack.

Fails if either of `x` or `x+1` is not a valid [VM number](#vm-number).


#### 1SUB

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x8c  | (x → x–1)       | 2; [standard memory cost](#standard-memory-cost)

Pops a [number](#vm-number) from the data stack, subtracts 1 from it, and pushes the result to the data stack.

Fails if either of `x` or `x-1` is not a valid [VM number](#vm-number).


#### NEGATE

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x8f  | (x → –x)        | 2; [standard memory cost](#standard-memory-cost)

Pops a [number](#vm-number) from the data stack, negates it, and pushes the result to the data stack.

Fails if either of `x` or `-x` is not a valid [VM number](#vm-number).

#### ABS

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x90  | (x → abs(x))    | 2; [standard memory cost](#standard-memory-cost)

Pops a [number](#vm-number) from the data stack, negates it if it is less than 0, and pushes the result to the data stack.

Fails if either of `x` or `abs(x)` is not a valid [VM number](#vm-number).


#### NOT

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x91  | (p → ~p)    | 2; [standard memory cost](#standard-memory-cost)

Pops a [boolean](#vm-boolean) from the data stack, negates it, and pushes the result to the data stack.


#### 0NOTEQUAL

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x92  | (x → x ≠ 0)     | 2; [standard memory cost](#standard-memory-cost)

Pops a [number](#vm-number) from the data stack, and results in `false` if the number is equal to 0 and `true` otherwise. Pushes the result to the data stack.

Fails if `x` is not a valid [VM number](#vm-number).


#### ADD

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x93  | (x y → x+y)     | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, adds them, and pushes the result to the data stack.

Fails if any of `x`, `y`, or `x+y` is not a valid [VM number](#vm-number).


#### SUB

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x94  | (x y → x–y)     | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, takes their difference, and pushes the result to the data stack.

Fails if any of `x`, `y`, or `x-y` is not a valid [VM number](#vm-number).


#### MUL

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x95  | (x y → x·y)     | 8; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, multiplies them, and pushes the result to the data stack.

Fails if any of `x`, `y`, or `x·y` is not a valid [VM number](#vm-number).


#### DIV

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x96  | (x y → x/y)     | 8; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, divides them rounding toward zero to an integer, and pushes the result to the data stack.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number), or if `y` is zero.


#### MOD

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x97  | (x y → x mod y) | 8; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from their data stack, determines the remainder of `x` divided by `y`, and pushes the result to the data stack. A non-zero result has the same sign as the divisor.

Example     | Result
------------|--------
12 mod 10   | 2
–12 mod 10  | 8
12 mod –10  | –8
–12 mod –10 | –2

Fails if either of `x` or `y` is not a valid [VM number](#vm-number), or if `y` is zero.



#### LSHIFT

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x98  | (x y → x << y)  | 8; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, multiplies `x` by `2**y` (i.e., an arithmetic left shift with sign extension), coerces the result to a [string](#vm-string), and pushes it to the data stack.

Example     | Result
------------|--------
5 << 1      | 10
5 << 2      | 20
-5 << 1     | -10

Fails if any of `x`, `y` or `x * 2**y`is not a valid [VM number](#vm-number), or if `y` is less than zero.


#### RSHIFT

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x99  | (x y → x >> y)  | 8; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, divides `x` by `2**y` rounding to an integer toward negative infinity (i.e., an arithmetic right shift with sign extension), and pushes the result to the stack.

Example     | Result
------------|--------
10 >> 1     | 5
10 >> 2     | 2
1 >> 1      | 0
-1 >> 1     | -1
-10 >> 2    | -3

Fails if either of `x` or `y` is not a valid [VM number](#vm-number), or if `y` is less than zero.



#### BOOLAND

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x9a  | (p q → p && q)  | 2; [standard memory cost](#standard-memory-cost)

Pops two [booleans](#vm-boolean) from the data stack. Pushes `true` to the data stack if both are `true`, and pushes `false` otherwise.


#### BOOLOR

Code  | Stack Diagram     | Cost
------|-------------------|----------------------------
0x9b  | (p q → p \|\| q)  | 2; [standard memory cost](#standard-memory-cost)

Pops two [booleans](#vm-boolean) from the data stack. Pushes `false` to the data stack if both are `false` and pushes `true` otherwise.


#### NUMEQUAL

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x9c  | (x y → x == y)  | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack. Pushes `true` to the data stack if they are equal and pushes `false` otherwise.

Note that two strings representing the same number may differ due to redundant leading zeros.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number).


#### NUMEQUALVERIFY

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x9d  | (x y → ∅)       | 2; [standard memory cost](#standard-memory-cost)

Equivalent to [NUMEQUAL](#numequal) [VERIFY](#verify).

Pops two [numbers](#vm-number) from the data stack, and fails if they are not equal.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number), or if they are not numerically equal.


#### NUMNOTEQUAL

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x9e  | (x y → x ≠ y)   | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, results in `false` if they are equal and in `true` otherwise, and pushes the result to the data stack.

Note that two strings representing the same number may differ due to redundant leading zeros.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number).


#### LESSTHAN

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0x9f  | (x y → x < y)   | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, results in `true` if `x` is less than `y` and `false` otherwise, and pushes the result to the data stack.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number).


#### GREATERTHAN

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0xa0  | (x y → x > y)   | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, results in `true` if `x` is greater than `y` and in `false` otherwise, and pushes the result to the data stack.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number).


#### LESSTHANOREQUAL

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0xa1  | (x y → x ≤ y)   | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, results in `true` if `x` is less than or equal to `y` and in `false` otherwise, and pushes the result to the data stack.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number).

#### GREATERTHANOREQUAL

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0xa2  | (x y → x ≥ y)   | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, results in `true` if `x` is greater than or equal to `y` and in `false` otherwise, and pushes the result to the data stack.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number).


#### MIN

Code  | Stack Diagram    | Cost
------|------------------|----------------------------
0xa3  | (x y → min(x,y)) | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the data stack, results in `x` if `x` is less than or equal to `y` and in `y` otherwise, and pushes the result to the data stack.

Fails if either of `x` or `y` is not a valid [VM number](#vm-number).


#### MAX

Code  | Stack Diagram    | Cost
------|------------------|----------------------------
0xa4  | (x y → max(x,y)) | 2; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the stack, results in `x` if `x` is greater than or equal to `y` and in `y` otherwise, and pushes the result to the data stack.

Fails if any of `x`, `y`, or `z` is not a valid [VM number](#vm-number).


#### WITHIN

Code  | Stack Diagram               | Cost
------|-----------------------------|----------------------------
0xa5  | (x y z → y ≤ x < z) | 4; [standard memory cost](#standard-memory-cost)

Pops two [numbers](#vm-number) from the stack, results in [true](#vm-boolean) if `x` is greater or equal to the mininum value `y` and less than the maximum value `z`, and pushes the result to the stack.

Fails if any of `x`, `y`, or `z` is not a valid [VM number](#vm-number).


### Cryptographic instructions


#### SHA256

Code  | Stack Diagram                  | Cost
------|--------------------------------|-----------------------------------------------------
0xa8  | (a → SHA-256(a))               | max(64, 4·L<sub>a</sub>) + [standard memory cost](#standard-memory-cost)

Replaces top stack item with its [SHA-256](https://en.wikipedia.org/wiki/SHA-2) hash value.


#### SHA3

Code  | Stack Diagram                  | Cost
------|--------------------------------|-----------------------------------------------------
0xaa  | (a → SHA3-256(a))              | max(64, 4·L<sub>a</sub>) + [standard memory cost](#standard-memory-cost)

Replaces top stack item with its [SHA3-256](blockchain.md#sha3) hash value.


#### CHECKSIG

Code  | Stack Diagram                  | Cost
------|--------------------------------|-----------------------------------------------------
0xac  | (sig hash pubkey → q)          | 1024; [standard memory cost](#standard-memory-cost)

Pops the top three items on the data stack, verifies the [signature](blockchain.md#signature) `sig` of the `hash` with a given public key `pubkey` and pushes `true` if the signature is valid; pushes `false` if it is not.

Fails if `hash` is not a 32-byte string.


#### CHECKMULTISIG

Code  | Stack Diagram                  | Cost
------|--------------------------------|-----------------------------------------------------
0xad  | (sig<sub>m-1</sub> ... sig<sub>0</sub> hash pubkey<sub>n-1</sub> ... pubkey<sub>0</sub> m n → q)  | 1024·n; [standard memory cost](#standard-memory-cost)

1. Pops non-negative [numbers](#vm-number) `n` and `m` from the data stack.
2. If `n` is positive, verifies that `m` is also positive.
3. Pops `n` public keys.
4. Pops `hash` from the data stack.
5. Pops `m` signatures.
6. Verifies [signatures](blockchain.md#signature) one by one against the public keys and the given `hash`. Signatures must be in the same order as public keys and no two signatures are verified with the same public key.
7. Pushes `true` if all of the signatures are valid, and `false` otherwise.

Failure conditions:

* there are fewer than `m+n+3` items on the data stack, or
* the top item (`n`) is not a number, or
* `n` is negative, or
* the second from the top item (`m`) is not a number, or
* `m` is negative, or
* `m` is greater than `n`, or
* `m` is zero and `n` is positive.



#### TXSIGHASH

Code  | Stack Diagram                  | Cost
------|--------------------------------|-----------------------------------------------------
0xae  | (∅ → hash)                     | 256 + [standard memory cost](#standard-memory-cost)

Computes the transaction signature hash corresponding to the current entry. Equals [SHA3-256](blockchain.md#sha3) of the concatenation of the current [entry ID](blockchain.md#entry-id) and [transaction ID](blockchain.md#transaction-id):

    TXSIGHASH = SHA3-256(entryID || txID)

This instruction is typically used with [CHECKSIG](#checksig) or [CHECKMULTISIG](#checkmultisig).

Fails if executed in the [block context](#block-context).


#### BLOCKHASH

Code  | Stack Diagram                  | Cost
------|--------------------------------|-----------------------------------------------------
0xaf  | (∅ → hash)                     | 1 + [standard memory cost](#standard-memory-cost)

Returns the [block ID](blockchain.md#block-id).

Typically used with [CHECKSIG](#checksig) or [CHECKMULTISIG](#checkmultisig).

Fails if executed in the [transaction context](#transaction-context).





### Introspection instructions

The following instructions are defined within a [transaction context](#execution-context). In the block context these instructions cause VM to halt immediately and return false.

Note: [standard memory cost](#standard-memory-cost) is applied *after* the instruction is executed in order to determine the exact size of the encoded data (this also applies to [ASSET](#asset), even though the result is always 32 bytes long).


#### CHECKOUTPUT

Code  | Stack Diagram                                        | Cost
------|------------------------------------------------------|-----------------------------------------------------
0xc1  | (index data amount assetid version prog → q)         | 16; [standard memory cost](#standard-memory-cost)

1. Pops 6 items from the data stack: `index`, `data`, `amount`, `assetid`, `version`, `prog`.
2. Fails if `index` is negative or not a valid [number](#vm-number).
3. Fails if the number of outputs is less or equal to `index`.
4. Fails if `amount` and `version` are not non-negative [numbers](#vm-number).
5. If the current entry is a [Mux](blockchain.md#mux-1):
    1. Finds a [destination entry](blockchain.md#value-destination-1) at the given `index`.
    2. If the entry satisfies all of the following conditions pushes [true](#vm-boolean) on the data stack; otherwise pushes [false](#vm-boolean):
        1. the destination entry is an [output](blockchain.md#output-1) or a [retirement](blockchain.md#retirement-1),
        2. if the destination is an output: control program equals `prog` and VM version equals `version`,
        3. if the destination is a retirement:
            * `version` must be 1,
            * `prog` must begin with a [FAIL](#fail) instruction.
        4. asset ID equals `assetid`,
        5. amount equals `amount`,
        6. `data` is an empty string or it matches the 32-byte data string in the destination entry.
6. If the entry is an [issuance](blockchain.md#issuance-1) or a [spend](blockchain.md#spend-1):
    1. If the [destination entry](blockchain.md#value-destination-1) is a [Mux](blockchain.md#mux-1), performs checks as described in step 5.
    2. If the [destination entry](blockchain.md#value-destination-1) is an [output](blockchain.md#output-1) or a [retirement](blockchain.md#retirement-1):
        1. If `index` is not zero, pushes [false](#vm-boolean) on the data stack.
        2. Otherwise, performs checks as described in step 5.2.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [mux](blockchain.md#mux-1), an [issuance](blockchain.md#issuance-1) or a [spend](blockchain.md#spend-1).

#### ASSET

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xc2  | (∅ → assetid)  | 1; [standard memory cost](#standard-memory-cost)

If the current entry is an [issuance](blockchain.md#issuance-1) or a [spend](blockchain.md#spend-1) entry, pushes the `SpentOutput.Source.Value.AssetID` of that entry.

If the current entry is a [nonce](blockchain.md#nonce) entry, verifies that the `AnchoredEntry` field is an [issuance](blockchain.md#issuance-1) entry, and pushes the `Value.AssetID` of that issuance entry. Fails if `AnchoredEntry` is not an issuance version 1.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [nonce](blockchain.md#nonce), an [issuance](blockchain.md#issuance-1) or a [spend](blockchain.md#spend-1).

#### AMOUNT

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xc3  | (∅ → amount)   | 1; [standard memory cost](#standard-memory-cost)

If the current entry is an [issuance](blockchain.md#issuance-1) or a [spend](blockchain.md#spend-1) entry, pushes the `SpentOutput.Source.Value.Amount` of that entry.

If the current entry is a [nonce](blockchain.md#nonce) entry, verifies that the `AnchoredEntry` field is an [issuance](blockchain.md#issuance-1) entry, and pushes the `Value.Amount` of that issuance entry. Fails if `AnchoredEntry` is not an issuance version 1.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [nonce](blockchain.md#nonce), an [issuance](blockchain.md#issuance-1) or a [spend](blockchain.md#spend-1).


#### PROGRAM

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xc4  | (∅ → program)   | 1; [standard memory cost](#standard-memory-cost)

1. In [transaction context](#transaction-context):
  * For [spends](blockchain.md#spend-1): pushes the control program from the output being spent.
  * For [issuances](blockchain.md#issuance-1): pushes the issuance program.
  * For [muxes](blockchain.md#mux-1): pushes the mux program.
  * For [nonces](blockchain.md#nonce): pushes the nonce program.
2. In [block context](#block-context):
  * Pushes the current [consensus program](blockchain.md#block-header) being executed (that is specified in the previous block header).


#### MINTIME

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xc5  | (∅ → timestamp) | 1; [standard memory cost](#standard-memory-cost)

Pushes the [transaction header](blockchain.md#transaction-header) mintime in milliseconds on the data stack.
If the value is greater than 2<sup>63</sup>–1, pushes 2<sup>63</sup>–1 (encoded as [VM number](#vm-number) 0xffffffffffffff7f).

Fails if executed in the [block context](#block-context).

#### MAXTIME

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xc6  | (∅ → timestamp) | 1; [standard memory cost](#standard-memory-cost)

Pushes the [transaction header](blockchain.md#transaction-header) maxtime in milliseconds on the data stack.
If the value is zero or greater than 2<sup>63</sup>–1, pushes 2<sup>63</sup>–1 (encoded as [VM number](#vm-number) 0xffffffffffffff7f).

Fails if executed in the [block context](#block-context).

#### TXDATA

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xc7  | (∅ → string32)  | 1; [standard memory cost](#standard-memory-cost)

Pushes the transaction's data string as stored in the [transaction header](blockchain.md#transaction-header).

Fails if executed in the [block context](#block-context).


#### ENTRYDATA

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xc8  | (∅ → string32)  | 1; [standard memory cost](#standard-memory-cost)

Pushes the data string as stored in the current [entry](blockchain.md#entry).

Fails if executed in the [block context](#block-context).

Fails if the current entry is not an [issuance](blockchain.md#issuance-1), a [spend](blockchain.md#spend-1), an [output](blockchain.md#output-1) or a [retirement](blockchain.md#retirement-1).


#### INDEX

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xc9  | (∅ → index)     | 1; [standard memory cost](#standard-memory-cost)

Pushes the [ValueDestination.position](blockchain.md#value-destination-1) of the current entry on the data stack.

Fails if executed in the [block context](#block-context).

Fails if the current entry is not an [issuance](blockchain.md#issuance-1) or a [spend](blockchain.md#spend-1).


#### ENTRYID

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xca  | (∅ → entryid)   | 1; [standard memory cost](#standard-memory-cost)

Pushes the [current entry ID](blockchain.md#entry-id) on the data stack (e.g. a [spend](blockchain.md#spend-1), an [issuance](blockchain.md#issuance-1) or a [nonce](blockchain.md#nonce)).

Fails if executed in the [block context](#block-context).


#### OUTPUTID

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xcb  | (∅ → outputid)  | 1; [standard memory cost](#standard-memory-cost)

Pushes the [spent output ID](blockchain.md#spend-1) on the data stack.

Fails if executed in the [block context](#block-context).

Fails if the current entry is not a [spend](blockchain.md#spend-1).


#### NONCE

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xcc  | (∅ → nonce)     | 1; [standard memory cost](#standard-memory-cost)

Pushes the [anchor ID](blockchain.md#issuance-1) of the [issuance entry](blockchain.md#issuance-1) on the data stack.

Fails if executed in the [block context](#block-context).

Fails if the current entry is not an [issuance](blockchain.md#issuance-1).


#### NEXTPROGRAM

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xcd  | (∅ → program)   | 1; [standard memory cost](#standard-memory-cost)

Pushes the [next consensus program](blockchain.md#block-header) specified in the current block header.

Fails if executed in the [transaction context](#transaction-context).


#### BLOCKTIME

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0xce  | (∅ → timestamp) | 1; [standard memory cost](#standard-memory-cost)

Pushes the block timestamp in milliseconds on the data stack.

Fails if executed in the [transaction context](#transaction-context).



### Expansion opcodes

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0x50, 0x61, 0x62, 0x65, 0x66, 0x67, 0x68, 0x8a, 0x8d, 0x8e, 0xa6, 0xa7, 0xa9, 0xab, 0xb0..0xbf, 0xcd..0xcf, 0xd0..0xff  | (∅ → ∅)     | 1

The unassigned codes are reserved for future expansion.

If the [expansion flag](#vm-state) is on, these opcodes have no effect on the state of the VM except from reducing run limit by 1 and incrementing the program counter.

If the [expansion flag](#vm-state) is off, these opcodes immediately fail the program when encountered during execution.





# References

* [FIPS180: "Secure Hash Standard", United States of America, National Institute of Standards and Technology, Federal Information Processing Standard 180-2](http://csrc.nist.gov/publications/fips/fips180-2/fips180-2withchangenotice.pdf).
* [FIPS202: Federal Inf. Process. Stds. (NIST FIPS) - 202 (SHA3)](https://dx.doi.org/10.6028/NIST.FIPS.202)
* [LEB128: Little-Endian Base-128 Encoding](https://developers.google.com/protocol-buffers/docs/encoding)
* [RFC 6962](https://tools.ietf.org/html/rfc6962#section-2.1)
* [RFC 8032](https://tools.ietf.org/html/rfc8032)
