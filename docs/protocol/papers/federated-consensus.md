# Federated Consensus

* [Introduction](#introduction)
* [Algorithm overview](#algorithm-overview)
* [Integrity guarantees](#integrity-guarantees)
* [Liveness guarantees](#liveness-guarantees)
* [Key management](#key-management)
* [Membership changes](#membership-changes)
* [Policy enforcement](#policy-enforcement)
* [Future improvements](#future-improvements)


## Introduction

Federated consensus is a mechanism ensuring that only one valid blockchain is published and therefore double-spends are prevented.

The consensus protocol is mostly distinct from other blockchain validation rules. Those specify _which_ blockchain is valid, while consensus makes sure there is no more than _just one_ valid blockchain.

In this guide we discuss the design of the federated consensus used in Chain Protocol: its goals, use cases, threat models, and areas for future improvement.


## Algorithm overview

A federation consists of a single _block generator_ and a group of _block signers_. 

The block generator:

* receives transactions from the network,
* filters out transactions that are invalid,
* filters out transactions that do not pass local policy checks,
* periodically aggregates them in a new block,
* sends each proposed block to the block signers for approval.

Each block signer:

* verifies that the proposed block is signed by the generator,
* verifies that it follows the protocol rules (excluding checks for block signers’ signatures, which are not yet present),
* verifies that it extends their chain (i.e. the block does not fork the chain),
* verifies that the block contains an acceptable _consensus program_ (for authenticating the next block),
* signs the block.

Once the block generator receives signatures from enough block signers (as defined by the block's consensus program), it publishes the block to the network. All network nodes validate that block on receiving it (including block signers’ signatures). Meanwhile the block generator assembles the next block with more transactions.

[sidenote]

For a detailed description of the consensus protocol, see the [Federated Consensus Protocol](../specifications/consensus.md) specification.

Chain VM supports block introspection instructions that allow custom consensus programs. See the [Chain VM](../specifications/vm1.md#block-context) specification for details.

[/sidenote]

Each block is authenticated to the network via a _consensus program_ declared in the previous block. The consensus program is a predicate that must be satisifed by _program arguments_ in the subsequent block. A typical consensus programs implements an “M-of-N multisignature” rule where M is the number of required signatures and N is the number of block signers. Public keys are included within the program and the signatures are provided in the arguments list. The program checks that the signatures are correctly made from the subsequent block and match the keys in the consensus program. New blocks may reuse the same consensus program or change it to a new one (as when members join and leave the federation) as long as a quorum of block signers approves the change.

The signature supplied by a block generator to authenticate a block-signing request to the block signers is distinct from the block-ratifying signature that the signers themselves supply. Only the latter are part of the rules used by the rest of the network to validate blocks. This allows block signers to evolve the consensus mechanism without any additional support from the rest of the network.


## Integrity guarantees

The Chain Protocol prescribes a number of context-free rules that define (in part) which transactions are valid - amounts must balance, signatures must be correct, etc. However, some validity rules depend on a wider context. Additional rules are necessary to ensure that:

* two transactions do not spend the same output,
* transactions are final,
* there is only one version of the blockchain.

To prevent double spending, each network node tracks a set of *unspent transaction outputs* (the “UTXO set”). Each transaction is allowed to spend only existing unspent outputs. When validating the transactions in each incoming block, the network node updates its UTXO set, removing spent outputs and adding new ones.

Network nodes also verify that each block correctly links to its predecessor and is correctly signed. If a node detects two different, correctly signed blocks at one point in history, it stops immediately and reports an integrity violation to the administrator. The node refuses to process transactions from either of the two blocks and waits for out-of-band resolution. Therefore network nodes may only experience double-spends or have their transactions reversed if both of these conditions are satisfied:

[sidenote]

Note that a node should fail-stop even if one of two blocks is correctly signed, but contains invalid transactions (e.g. double spends, or transactions not satisfying control and issuance programs). This provides protection for those using compact proofs. Such users rely on block signers’ signatures to verify that a certain transaction is included in the blockchain rather than on validating all transactions.

[/sidenote]

1. a quorum of block signers signs two different blocks with a common ancestor (“forks the blockchain”),
2. block signers perform a partition attack to prevent each part of the network seeing both blocks.

Honest block signers are therefore responsible for not signing two forks of the blockchain. They do so simply by refusing to sign an alternative version of a proposed block by a block generator. A dishonest block generator is not able to fork the blockchain. An attempt to do so may lead to a deadlock: block signers will not be able to reach a quorum and will need an out-of-band agreement about the block to finalize and publish.


## Liveness guarantees

Simplicity and performance of the consensus protocol comes with a liveness tradeoff. While the block generator is not capable of forking the blockchain, it does have control over network liveness: if the block generator crashes or otherwise stops producing new blocks, the blockchain halts. The block generator can also deadlock the network by sending inconsistent blocks to different block signers. Additionally, the block generator has control over the block timestamp, and can produce blocks with artificially “slow” timestamps.

Since the responsibility for preventing blockchain forks resides in the block signers, the block generator can be made highly available using traditional replication methods, without the need for a Byzantine-fault-tolerant agreement protocol.

A quorum of block signers can temporarily stop the network by refusing to sign new blocks. They can also permanently deadlock other nodes by attempting to fork a blockchain, provided these nodes receive blocks from both chains (i.e. if the network is not partitioned). If deadlock occurs among block signers or on the entire network level, it must be resolved manually using an out-of-band agreement.


## Key management

The block generator and block signers store signing keys in a hardware security module (HSM) that prevents leakage of long-term cryptographic material. If the HSM needs to be upgraded, or keys need to be rotated for any reason, the federation may agree out of band on a new consensus program and start using it in new blocks beginning at a certain timestamp.

The consensus program is evaluated by the Chain VM. By using introspection instructions such as [NEXTPROGRAM](../specifications/vm1.md#nextprogram) and [CHECKPREDICATE](../specifications/vm1.md#checkpredicate) directly inside the consensus program it is possible to create more sophisticated schemes such as temporary key delegation or automatic rotation.

The [Blockchain Programs](blockchain-programs.md) paper discusses in detail different ways to use programs and introspection instructions to build secure blockchain consensus schemes.


## Membership changes

Members can be added and removed from the federation of block signers using the same techniques as described in the key management session. As with any change to consensus program, it requires approval by a quorum of existing block signers.

Extra care must be taken when changing membership in order to avoid changes to liveness or integrity guarantees. For instance, if a member is removed from a 5-of-7 multisignature consensus program and the threshold is not lowered, the rule becomes 5-of-6 and the network can tolerate downtime of only one block signer instead of two. However, if the threshold is lowered to 4-of-6, then it can still tolerate two crashes, but only one byzantine failure among block signers. Generally, it is recommended always to maintain the stable federation size, especially if it is relatively small.


## Policy enforcement

The block generator may apply local policy to filter out non-compliant transactions. Since policy enforcement is not a part of the protocol rules, it is more flexible, can be changed at will, and may use confidential information that should not be shared with the whole network. The cost of this flexibility is lower security: if some transactions “slip through” one node’s filter, they are recorded forever in the ledger and additional measures limiting subsequent transactions are necessary to mitigate any potential damage.


## Future improvements

The consensus mechanism may be improved without disrupting the network. This section provides a brief overview of improvements that are desirable and may be introduced in future versions of the Chain Protocol and Chain Core software.


#### Double-phase commitment 

The current consensus mechanism does not allow block signers to enforce their own local policies and refuse to sign otherwise valid blocks. If a signer wants to re-negotiate the block content with a block generator, other signers who already signed the first version of that block cannot safely sign another version as this undermines integrity guarantees.

However, if block signers use two rounds of signing, with _private signatures_ first and then _public signatures_ after reaching quorum, then they are able to reject proposed blocks and re-sign alternative versions any number of times privately. A block signer’s _public signature_ could only be used on one block at each point in history.


#### Fraud proofs protocol

Nodes may implement stronger protection against blockchain forks by not relying exclusively on a quorum of block signers. In addition to existing fail-stop rules, nodes may communicate directly with other nodes using a peer-to-peer protocol to verify that they see the same chain of block headers. When a fork is detected, nodes let each other know about it. And if any given node cannot reach a well-known peer, it may pause in its processing of the blockchain assuming that a network partition attack could be under way. This makes such attacks more difficult: faulty block signers must isolate not one node, but a whole group of interconnected nodes to prevent them from learning about the existence of an alternative chain. 

#### Byzantine fault tolerance

A consensus mechanism based on a single block generator is not ideal for all scenarios. To improve liveness guarantees without compromising security, a more sophisticated byzantine agreement protocol is required. Existing proposals such as PBFT, Tendermint and Byzcoin demonstrate potential in this area.

#### Bitcoin checkpoints

The present consensus mechanism assumes that a quorum (majority) of federation members is honest and that their keys are well-protected (as do more sophisticated byzantine consensus protocols). However, if keys used in older blocks ever become compromised, it is possible to fork the blockchain at an arbitrary point in the past.

One way to offer long-term blockchain integrity is periodically to commit the latest block hash to the Bitcoin blockchain. This way, even a compromised quorum of keys cannot produce a valid fork without being detected by nodes cross-checking against the Bitcoin blockchain. With this technique, the network needs to trust the quorum of block signers for much less time to not fork the network: hours instead of years.


