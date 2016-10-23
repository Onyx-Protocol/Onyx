# Blockchain Programs

* [Introduction](#introduction)
* [Chain Virtual Machine](#chain-virtual-machine)
* [Ivy](#ivy)
* [Programs](#programs)
  * [Control programs](#control-programs)
  * [Issuance programs](#issuance-programs)
  * [Consensus programs](#consensus-programs)
* [Examples](#program-examples)
  * [Signature programs](#signature-programs)
  * [Offers](#offers)
  * [State machines](#state-machines)
  * [Singleton](#singleton)
  * [Private Contracts](#private-programs)


## Introduction

Chain Protocol enables flexible control over assets by supporting custom logic at three levels:

* **Issuance programs**, that specify the rules for issuing new units of an asset.
* **Control programs**, that specify the rules for spending existing units of an asset.
* **Consensus programs**, that specify the rules for accepting new blocks.

Each program authenticates the data structure in which it is used. Programs run deterministically, use capped memory and time requirements, and can be evaluated in parallel.

Programs are flexible enough to allow implementing:

* a wide range of financial instruments (such as options, bonds, and swaps), 
* sophisticated security schemes for holding assets,
* and “smart program” applications such as offers, order books, and auctions.

This document discusses design and use cases for custom programs on the blockchain.

## Chain Virtual Machine

A program is written in bytecode — instructions for the Chain Virtual Machine (CVM). The CVM is a stack machine: each instruction performs operations on a *data stack*, usually working on the items on top of the stack. All items on the data stack are strings of bytes, although some instructions convert them to and from numbers or booleans in order to perform operations on them. The CVM also has an *alt stack* to simplify stack manipulation.

[sidenote]

Bitcoin, similarly, uses programs as predicates in order to determine whether a given state transition — encoded in a transaction — is authorized. This is different from Ethereum’s approach, in which programs directly compute the resulting state.

[/sidenote]

### Run limit

The CVM’s instruction set is Turing complete. To prevent unbounded use of computational resources, the protocol allows networks to set a *run limit* that a program is not allowed to exceed. Each instruction consumes some of the limit as it runs, according to its *run cost*. Processing-intensive instructions, such as signature checks, are more expensive.

The run cost also takes into account the stack's current memory usage. Adding an item to the stack has a cost based on the size of the item; removing an item from the stack refunds that cost.

[sidenote]

Both Bitcoin and Ethereum have restrictions that prevent program execution from using excessive time or memory. Chain’s run limit mechanism is similar to Ethereum’s “gas,” except that there is no on-chain accounting for the execution cost of a transaction.

[/sidenote]

### Instruction Set

The CVM has some overlaps and similarities with Bitcoin Script, but adds instructions to support additional functionality, including loops, state transitions (through transaction introspection), and program evaluation.

What follows is a summary of the functionality provided by CVM instructions. For a complete list and more precise definitions, see the [VM specification](../specifications/vm1.md).

#### Stack manipulation

Programs may encode bytestrings to push on data stack using a range of `PUSHDATA` instructions. Instructions `DROP`, `DUP`, `SWAP`, `PICK` and others allow moving stack items around. More complex stack manipulations can be assisted by `TOALTSTACK` and `FROMALTSTACK` instructions that move items between the data stack and an auxiliary alt stack.

#### String manipulation

`EQUAL` checks for the equality of two strings. `CAT`, `SUBSTR`, `LEFT`, and `RIGHT` perform operations on strings from the top of the stack. `AND`, `OR`, `XOR` perform bitwise operations.

#### Arithmetic operations

While all items on the stack are strings, some instructions interpret items as numbers, using 64-bit two's complement representation.

CVM deterministically checks for overflows: if the result overflows (e.g. too large numbers are multiplied), execution immediately fails.


#### Boolean operations

Items on the stack can also be interpreted as booleans. Empty strings and strings consisting of zero bytes are coerced to `false`, all others are coerced to `true`.

#### Cryptographic operations

The `SHA256` and `SHA3` instructions execute corresponding hash functions and output 32-byte strings.

The `CHECKSIG` instruction checks the validity of an Ed25519 signature against a given public key and a message hash.

[sidenote]

While similar to Bitcoin instructions, `CHECKSIG` and `CHECKMULTISIG` are generalized to accept an arbitrary message hash. This enables integration with external authoritative data sources and, more importantly, [signature programs](#signature-programs) discussed below.

[/sidenote]

`CHECKMULTISIG` checks an “M-of-N” signing condition using `M` signatures and `N` public keys.

#### Control flow instructions

`VERIFY` pops the top value from the data stack and checks if it is `true`. If it is not, or if there is no top value, the entire program fails.

`JUMPIF` conditionally jumps to another part of the code, based on the current value on top of the stack. This can be used to implement conditionals and loops.

`CHECKPREDICATE` executes a program (written in CVM bytecode) in a separate VM instance. Nested executions are allowed, but the depth is capped by memory cost that is subtracted from the available run limit and refunded when the nested VM instance completes execution.

#### Introspection instructions

The CVM provides operations that, when used in a control or issuance program, introspect parts of a transaction attempting to spend that output. 

[sidenote]

The EVM includes many opcodes that provide introspection into the execution environment, including the global mutable state.

In contrast, CVM allows introspection only of the immutable data declared in the transaction, similar to Bitcoin’s `CHECKLOCKTIMEVERIFY` and `CHECKSEQUENCEVERIFY` instructions that check absolute and relative transaction lock times, respectively.

[/sidenote]

`CHECKOUTPUT` allows an input to introspect the outputs of the transaction. This allows it to place restrictions on how the input values are subsequently used. This instruction provides functionality similar to the `CHECKOUTPUTVERIFY` opcode proposed by Malte Möser, Ittay Eyal, and Emin Gün Sirer in their [Bitcoin Covenants](http://fc16.ifca.ai/bitcoin/papers/MES16.pdf) paper. `CHECKOUTPUT` also allows implementing arbitrary state-machines within a UTXO model as was proposed earlier by Oleg Andreev in [Pay-to-Contract](https://github.com/oleganza/bitcoin-papers/blob/master/SmartContractsSoftFork.md) paper.

`MINTIME` and `MAXTIME` allow limitations on when a UTXO can be spent. `AMOUNT`, `ASSET`, `PROGRAM`, `REFDATAHASH`, and `INDEX` allow a control program to introspect the input itself.

## Ivy

Chain is developing a high-level language, *Ivy*, that compiles to CVM bytecode, to make it easier to write programs. Ivy is still evolving, and this explanation and tutorial is provided only to help ground the examples used below. Some of the compiled bytecode programs are tweaked for clarity or simplicity.

[sidenote]

Similarly, most development for the EVM is done using [Solidity](https://solidity.readthedocs.io/en/develop/), a high-level language that has been compared to JavaScript. While Ivy and Solidity have some similarities in syntax, they have very different semantics. Solidity can be roughly classified as an object-oriented imperative language, while Ivy fits better into the paradigm of a *declarative language*. This reflects the design differences between Ethereum's and Chain's transaction models. 

[/sidenote]

### Condition

A *condition* in Ivy is a statement that either *fails* or *succeeds*. Each condition begins with `verify` and ends with a semicolon.

`verify 1 + 1 == 2;`

This condition would compile to the CVM bytecode: `1 1 + 2 EQUAL VERIFY`.

A condition fails if the given expression evaluates to `false`. 

Ivy supports the same arithmetic, logical, cryptographic, and string operations as the CVM, but uses more familiar infix and function-call syntax:

`verify (5 > 4) && (2 < 3);` -> `5 4 GREATERTHAN 2 3 LESSTHAN VERIFY`
`verify sha3("hello world") == 0x644bcc7e564373040999aac89e7622f3ca71fba1d972fd94a31c3bfbf24e3938` -> `PUSHDATA(0x68656c6c6f20776f726c64) SHA3 PUSHDATA(0x644bcc7e564373040999aac89e7622f3ca71fba1d972fd94a31c3bfbf24e3938) VERIFY`

### Programs

*Programs* are an easy way to construct and combine sequences of conditions. Like conditions, programs do not change state or return values; they simply succeed or fail.

Here is an example of a control program written in Ivy:

	program SignatureControlProgram(publicKey) {
		path spend(signature) {
			verify checksig(publicKey, tx.hash, signature);
		}
	}

Let's break this program down piece by piece.

Programs can take *parameters*. This program takes one parameter, `publicKey`. Parameters are specified at the time the program is *instantiated*, or created. In the case of a control program like this, that is the time that a UTXO is added to the blockchain by a transaction.

Programs define one or more *paths*. This program has only one path: `spend`. If this control program could be spent in different ways, it would have more than one path. In the case of a control program like this, that is the time the UTXO is used as an input in a new transaction. 

Each path can take *arguments*. Arguments are provided in the input witness. Arguments are passed at the time the program is satisfied . This program takes one argument: `signature`.

Paths contain one or more conditions. This path only uses a single condition, which uses the `CHECKSIG` instruction to check that the provided signature on the hash of the new transaction corresponds to the previously specified public key.

Control and issuance programs have access to a global `tx` variable, which allows them to use the transaction introspection instructions. In this case, `tx.hash` uses the `TXSIGHASH` instruction to get the hash of the new transaction.

    TXSIGHASH SWAP CHECKSIG VERIFY

When the program is instantiated with a `publicKey` value — to be used in a UTXO — the compiler prepends an instruction pushing that value. For example, if the public key used to initialize it is `0xd75a98...`, the script becomes:

	PUSHDATA(0xd75a98...) TXSIGHASH SWAP CHECKSIG VERIFY

When the UTXO is spent and the control program is run, the virtual machine first takes arguments specified in the input witness and pushes them to the stack. In this case, that argument is a signature. The program then executes, first pushing the public key and then the transaction hash to the stack. The public key and transaction hash are then swapped to put them in the correct order for the next instruction, which pops all three items off the stack to check the signature, pushing `true` or `false` to the stack. `VERIFY` then pops the top value from the stack, and causes the program to fail if the value is `false`. (In an actual control program, the `VERIFY` instruction of the last condition in a path is omitted, since it is performed by the VM itself.)

Many control, issuance, and consensus programs use a multisignature check.

	program MultisigControlProgram(n, m, ...publicKeys[n]) {
		path spend(...signatures[m]) {
			verify checkmultisig(n, m, publicKeys..., tx.hash, signatures...);
		}
	}

The `...publicKeys[n]` syntax allows programs to take variable numbers of parameters or arguments.

### Evaluation

Normally, when a control program is added to the blockchain, the logic is available immediately. What if we don't want to reveal our public keys or logic when the control program is first put on the blockchain, but only when it is spent? The control program could commit to a *hash* of the relevant program. (Bitcoin supports a similar pattern, known as "Pay to Script Hash"; see [BIP 13](https://github.com/bitcoin/bips/blob/master/bip-0013.mediawiki)).

	program P2SHControlProgram(programHash) {
		path spend(program, m, ...args[m]) {
			verify sha3(program) == programHash;
			verify program(args...);
		}
	}

The program format is a useful tool for describing and developing generic patterns for control programs (and as a result is used throughout the rest of this guide). 

Programs *themselves* can instantiate programs with parameters to create new programs. In combination with output introspection, this allows construction of complex state machines.

This is examined in more detail in the [examples](#examples) below.

## Programs

On the high level, Chain Protocol uses three kinds of programs: control programs, issuance programs, and consensus program.

### Control programs

Control programs define the conditions for spending assets on a blockchain.

Control programs are specified in a transaction output, which also specifies an asset ID and amount. That value is stored on the blockchain in an unspent transaction output (UTXO). To spend that value, someone can create a transaction that uses that UTXO as the source of one of its inputs.

An example control program is described above.

### Issuance programs

Issuance programs define the rules for issuing new units of an asset onto the blockchain.

The issuance program for a given type of asset is fixed when the asset ID is first defined. The issuance program is part of the data structure hashed to create the asset ID, and therefore cannot be changed.

To issue units of an asset, an issuer creates a transaction with one or more issuance inputs specifying some amount of that asset to be issued. Arguments can be passed in the input witness.

A simple issuance program might just check one or more signatures on the transaction doing the issuance. It would therefore look a lot like the control program described above:

    program MultisigIssuanceProgram(n, m, ...publickeys[n]) {
    	path issue(...signatures[m]) {
    		verify checksig(publicKeys..., tx.hash, signatures...);
    	}
    }

### Consensus programs

Consensus programs define the rules for accepting a new block. 

Each block includes the consensus program that must be satisfied by the *next* block.

Chain's [federated consensus protocol](federated-consensus.md) relies on a quorum of block signers signing the hash of the block. The consensus program can therefore look a lot like the multisignature issuance and control programs described above:

    program ConsensusProgram(n, m, ...publickeys[n]) {
    	path checkBlock(...signatures[m]) {
    		verify checkmultisig(n, m, publicKeys..., block.hash, signatures...);
    	}
    }

### Signature programs

Chain's programming language also enables a powerful new way to authorize transactions. 

In the above examples of control programs and issuance programs, coinholders and issuers authorize transactions by signing a hash that commits to the entire transaction. This is the typical way that authorization works in UTXO-based cryptocurrencies such as Bitcoin.

[sidenote]

Bitcoin technically allows different "signature hash types" that could provide some of the functionality described below. These signature types are relatively inflexible and complex, and are rarely used in practice.

[/sidenote]

Signing the entire transaction hash is fine if you only want to authorize an input to be spent in a particular transaction. However, what if you only know or care about a particular part of a transaction at the time you sign it? 

For example, suppose Alice wants to sell 5 shares of Acme to Bob, in exchange for 10 USD. Alice wants to authorize the transfer of her Acme shares if and only if she receive payment of 5 USD to her own address. However, Alice does not care what the other input in the transaction will be — i.e., where the other payment will come from. If Alice **(Oleg: this sentence is incomplete.)**

Instead of authorizing a specific transaction, it would be useful if a spender or issuer could authorize any transaction that meets certain criteria.

To enable this, the control program for Alice's Acme shares cannot have the simple form described above, which checks a signature against the transaction hash. Instead, it should look like this:

    program P2SPControlProgram(publicKey) {
    	path spend(signature, program, m, ...args[m]) {
    		verify checksig(publicKey, program, signature)
    		verify program(args...);
    	}
    }

Instead of providing a signature of the transaction hash, the spender provides a signature of a particular *program*, which is then evaluated (with any given arguments). The combined signature and program are referred to as a *signature program*.

The signature program can use transaction introspection to set conditions on particular parts of the transaction.

For example:

    program SimpleSignatureProgram(targetHash) {
    	path check() {
    		verify tx.hash == targetHash;
    	}
    }

This program turns a signature program into a traditional signature by committing to a specific transaction hash.

But a signature program can do much more than that. For example, this program solves the "exchange" problem described above:

    program CheckOutputSignatureProgram(targetOutputIndex, targetAmount, targetAssetID, targetControlProgram, targetReferenceData) {
    	path default() {
    		verify tx.outputs[targetOutputIndex] == (targetAmount, targetAssetID, targetControlProgram, targetReferenceData);
    	}
    }

If this program is initialized with the details of the desired output — say, 5 USD sent to Alice's new address — and signed with the private key corresponding to Alice's , the combined signature program will authorize Alice's input to be spent only in a transaction that includes the desired output. 

[sidenote]

Christopher Allen and Shannon Appelcline explore ideas similar to signature programs in their working paper on “[smart signatures](https://github.com/WebOfTrustInfo/ID2020DesignWorkshop/blob/master/draft-documents/smarter-signatures.md).”

[/sidenote]

## Contract Examples

These examples are provided as illustrations only. They elide over some subtleties, and should not be considered final or secure. If you are interested in using Chain programs in production, contact us at hello@chain.com.

### Offers and order books

    program Offer(askingPrice, currency, sellerAddress) {
    	path lift(paymentIndex) {
    		verify tx.outputs[paymentIndex] == (askingPrice, currency, sellerContract);
    	}
    }

That program will be on the blockchain until someone satisfies it. What if we want to make it cancellable by the seller?

    program RevocableOffer(askingPrice, currency, sellerContract) {
    	path lift(paymentIndex) {
    		verify tx.outputs[paymentIndex] == (askingPrice, currency, sellerContract);
    	}
        
    	path cancel(m, ...args[m]) {
    		verify sellerContract(args...);
    	}
    }

What if we want the offer to be irrevocable for a certain period of time, and then automatically expire after some later point?

    program TimeLimitedOffer(askingPrice, currency, sellerContract, revocabilityTime, expirationTime) {
    	path lift(paymentIndex) {
    		verify maxtime < expirationTime;
    		verify tx.outputs[paymentIndex] == (askingPrice, currency, sellerContract);
    	}
        
    	path cancel(m, ...args[m]) {
    		verify mintime > revocabilityTime;
    		verify sellerContract(args...);
    	}
    }

What if we want to be able to fill a *partial* order, allowing someone to pay for part of the program and leaving the rest available for someone else to purchase?

    program PartiallyFillableOffer(pricePerUnit, currency, sellerContract, revocabilityTime, expirationTime) {
    	path lift(purchasedAmount, paymentIndex, remainderIndex) {
    		verify purchasedAmount > 0;
    		verify maxtime < expirationTime;
    		verify tx.outputs[paymentIndex] == (purchasedAmount * pricePerUnit, currency, sellerContract);
    		verify tx.outputs[remainderIndex] == (tx.currentInput.amount - purchasedAmount, 
    											  tx.currentInput.asset,
    											  tx.currentInput.program);
    	}
        
    	path cancel(m, ...args[m]) {
    		verify mintime > revocabilityTime;
    		verify sellerContract(args...);
    	}
    }

Notice that the remainder must be sent to a new program that is a duplicate of the current one, just controlling fewer assets.

### State Machines

What if you want to get more complex than just replicating the same program, but want to change its state when you do? This is where the program model really shines. Programs can instantiate programs with new parameters on the fly.

This program will prevent its assets from being transferred more than once within a certain time period:

    program OncePerPeriod(authorizationPredicate, lastSpend, period) {
    	path spend(m, ...args[m]) {
    		// check that the spending is otherwise authorized
    		// this could be a signature check
    		verify authorizationPredicate(args...);
            
    		// check that at least one day has passed
    		verify tx.mintime > lastSpend;
            
    		nextControlProgram = OncePerPeriod(authorizationPredicate,
    										   tx.maxtime,
    										   period);
                                               
    		verify tx.outputs[m] == (tx.currentInput.amount,
    								 tx.currentInput.asset,
    								 nextControlProgram);
    	}
    }


### Singletons

While most state should be tracked locally in the program parameters for a specific UTXO, some on-chain use cases may require keeping track of "global" state. For example, one may want to limit issuance of an asset, so only 100 units can be issued per day. This can be done using the *singleton* design pattern.

First, one needs to create an asset for which only one unit can ever be issued. This requires some understanding of how the Chain Protocol handles issuances. Unique issuance — ensuring that issuances cannot be replayed — is a challenging problem that is outside the scope of this paper. The Chain Protocol's solution is that each issuance input has a nonce, which, when combined with the transaction's mintime and maxtime and the asset ID, must be unique throughout the blockchain's history. As a result, an issuance program can ensure that it is only used once by committing to a specific nonce, transaction mintime, and transaction maxtime:

    program SinglyIssuableAssetSingletonToken(nonce, mintime, maxtime, amount, lockProgram) {
    	path issue(outputIndex) {
    		// ensure that asset can only be issued once
    		verify nonce == tx.currentInput.nonce;
    		verify mintime == tx.mintime;
    		verify maxtime == tx.maxtime;
            
    		// ensure that only one unit is issued
    		verify tx.currentInput.amount == 1;
            
    		// ensure that the issued unit is locked with the target lockProgram
    		verify tx.outputs[outputIndex] == (tx.currentInput.amount,
    										   tx.currentInput.asset,
    										   lockProgram)
    	}
    }

This means there will only be one UTXO on the blockchain with this asset ID at a given time. As a result, it can be used as a *singleton* — a token to keep track of some piece of global state for other asset IDs.

The `lockProgram` parameter of this program determines the rules that will govern the token.

For example, we've already seen the `OncePerPeriod` program. If that program is used as the “lock program”, the singleton token can be prevented from being spent more than once in a particular amount of time.

How does that help us with metered issuance? We can create a separate asset with an issuance program that checks that the singleton is also spent in the same transaction, and that no more than a given amount is issued.

    program MeteredAssetIssuanceProgram(authorizationPredicate, singletonAssetID, maxAmount) {
    	path issue(singletonControlProgram,
    			     singletonIndex, m, ...args[m]) {
    		// check that the issuance is otherwise authorized
    		verify authorizationPredicate(args...);
            
    		// check that no more than the max amount is being issued
    		verify tx.currentInput.amount < maxAmount;
            
    		// check that the singleton token is being spent
    		// its index and control program don't need to be checked
    		// which is why they are passed as arguments
    		verify tx.outputs[outputIndex] == (1,
    										   singletonAssetID,
    										   singletonControlProgram);
    	}
    }

### Private Contracts

What if parties want to avoid *ever* revealing the logic to the blockchain? They can avoid doing so — in the normal case — by adding an additional path that lets all interested parties spend the output without revealing the program:

    program PrivateContractControlProgram(programHash, n, ...publicKeys[n]) {
    	path settle(...sigs[n]) {
    		// all interested parties can agree to the final result of the program
    		verify checkmultisig(n, n, publicKeys..., tx.hash, sigs...);
    	}
        
    	path enforce(program, m, ...args[m]) {
    		// any party can reveal the program and enforce it
    		verify sha3(program) == programHash;
    		verify program(args...);
    	}
    }

Parties can evaluate the program offline, determine the result, mutually agree to how it should resolve, and provide their signatures on the resulting transaction. If any party refuses to agree to the result, another party can enforce the program by making its code public. This is more similar to how program enforcement works in the real world.

[sidenote]

This idea can be extended to implement full [Merklized Abstract Syntax Trees](http://www.mit.edu/~jlrubin/public/pdfs/858report.pdf) — programs for which unexecuted branches do not need to be revealed. Similar ideas have also been explored by so-called "payment channels" in Bitcoin, most famously in the [Lightning Network](https://lightning.network/) project, as well as "[state channels](http://www.jeffcoleman.ca/state-channels/)" in Ethereum.

[/sidenote]


