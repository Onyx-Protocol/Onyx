# Federated Consensus

* [Introduction](#introduction)
* [Consensus programs](#consensus-programs)
* [Federated consensus program](#federated-consensus-program)
* [Consensus algorithm](#consensus-algorithm)
* [Safety guarantees](#safety-guarantees)
* [Liveness guarantees](#liveness-guarantees)
* [Consensus program changes](#consensus-program-changes)
* [Policy enforcement](#policy-enforcement)
* [Future improvements](#future-improvements)

## Introduction

In this guide we discuss the design of the federated consensus protocol used by the Chain Protocol: its goals, use cases, threat models, and areas for future improvement.

Federated consensus is a mechanism ensuring that all participants in a network agree on a single transaction log. This prevents different versions of the ledger being shown to different participants — thus preventing “double-spending” of assets — as well preventing history from being edited. While the blockchain validation rules specify _whether_ a given blockchain is valid, the consensus protocol makes sure there is _only one_ valid blockchain on a given network. 

This consensus protocol is designed to be practically useful under a certain set of requirements and assumptions commonly encountered in permissioned blockchain networks. The Chain Protocol is capable of supporting alternative consensus protocols.

For a detailed description of the federated consensus protocol, see the [formal specification](../specifications/consensus.md).

## Consensus programs

The Chain Protocol blockchain validation rules are intentionally agnostic as to what kind of consensus protocol is enforced. Additionally, they do not play a role in the process by which consensus is reached. Instead, blockchains provide a way for network participants to evaluate whether consensus has been reached: namely, _consensus programs_. The consensus program specifies a set of conditions that must be satisfied for a block to be accepted. The separation of consensus logic from blockchain validation rules, together with flexibility of consensus programs, allows networks to adopt of arbitrary consensus protocols, even including ones based on proof-of-work and proof-of-stake.

The consensus program for each block is specified in the header of the previous block. When the block is validated by a network participant, the consensus program is executed with the *arguments* that are specified in the block header. If the consensus program fails, the block is considered invalid.

Consensus programs are written for and executed by the Chain Virtual Machine. See [Blockchain Programs](./blockchain-programs.md#consensus-programs) and [Chain VM Specification](../specifications/vm1.md) for details.

The consensus protocol presented here uses relatively simple consensus programs, but the Chain Protocol supports more complex programs that could allocate signing authority in more complex ways.


## Federated consensus program

Under the federated consensus protocol, blocks are considered accepted once they have been signed by a specified quorum of *block signers*. This is implemented using a consensus program that checks an “M-of-N multisignature” rule, where N is the number of block signers, and M is the number of signatures required for a block to be accepted. 

The program specifies the public keys of each of the N block signers. The signatures are passed in the arguments. The program checks the signatures against the prespecified public keys to confirm that they are valid signatures of the hash of the new block. New blocks may reuse the same consensus program or change it to a new one (as when members join and leave the federation) as long as a quorum of block signers approves the change.

The values of these parameters, M and N, can be tweaked according to business requirements or the security parameters of the network. The safety and liveness implications of these choices are described below.

## Consensus algorithm

Assuming that every participant in the network trusts a sufficient subset of the block signers, the consensus program described above reduces the problem of reaching network-wide consensus to the simpler problem of reaching a consensus of at least M out of N block signers.

To efficiently do so, block signers agree on a single *block generator*. The generator's signature is only used to coordinate the block signers; it is not seen or validated by the network. This allows block signers to evolve the consensus mechanism without any additional support from the rest of the network.

The block generator:

* receives transactions from the network,
* filters out transactions that are invalid,
* filters out transactions that do not pass local policy checks,
* periodically aggregates transactions into blocks,
* sends each proposed block to the block signers for approval.

Each block signer:

* verifies that the proposed block is signed by the generator,
* verifies that the block is valid under its (the signer’s) current state without checking that the previous _consensus program_ is satisfied (which requires the block signers’ signatures),
* verifies that it has not signed a different block at the same or greater height,
* verifies that the block timestamp is no more than 2 minutes ahead of the current system time,
* verifies that the block contains an acceptable consensus program (for authenticating the next block),
* if all checks passed, signs the block.

Once the block generator receives signatures from enough block signers (as defined by the previous block's consensus program), it publishes the block to the network. All network participants, including the block signers, validate that block (including checking the previous consensus program is satisfied) and update their state.


## Safety guarantees

Safety — specifically, the assurance that a valid block is the only valid block at that height — is guaranteed as long as no more than `2M - N - 1` block signers violate the protocol. For example, if the consensus program requires signatures by 3 of 4 block signers, the blockchain could only be forked if 2 block signers misbehave by signing two blocks at the same height.

An attempt to fork the blockchain by signing multiple blocks can be detected and proven by anyone with access to both block headers. Network participants could implement a gossip protocol to share block headers and detect an attempted fork immediately, as long as the network is not partitioned. Double-signing can be proven after the fact, and dealt with out-of-band.

Another potentially desirable safety guarantee is the assurance that, if a block header's consensus program is satisfied, there is a corresponding valid blockchain history — i.e., no transaction is unbalanced and all programmed conditions are satisfied. This guarantee is assured as long as no more than `N - M` block signers are faulty.

This latter guarantee is useful for clients that do not have full visibility into the blockchain, instead using _compact proofs_ to check some transaction or piece of state against the block headers. Network participants that have visibility into entire blocks and has seen the entire history of the blockchain — or which trust some other party which does have such visibility — have no need to trust block signers for this particular guarantee. 

## Liveness guarantees

Liveness is guaranteed as long as: 

* The generator is not faulty, and
* No more than `N - M` block signers are faulty

If the block generator crashes or becomes unavailable, the network cannot generate new blocks. If the block generator disobeys the protocol by sending different blocks to different signers, it can even *deadlock* the protocol.

This reliance on a single specific participant is relatively unusual for modern consensus protocols. However, it provides many efficiency and simplicity benefits, and has only limited downside in many target use cases.

Guaranteeing the liveness of the generator is an easier technical problem than a more complicated consensus protocol that allows any block signer to propose a block. The block generator can be operated as a distributed system within a single organization's trust boundary. It can therefore be made highly available using traditional replication methods, including non-Byzantine-fault-tolerant consensus protocols.

If the generator is intentionally shut down by the network operator, or if it ceases to correctly follow the protocol (whether due to a hack, a bug, or malicious intent by its operator), the network can safely halt until manual intervention.

## Consensus program changes

If members need to be added or removed from the federation, or if keys need to be rotated, the federation may agree out of band on a new consensus program to be used in new blocks beginning at a certain timestamp.

## Policy enforcement

The block generator may enforce local “policies” to filter out non-compliant transactions. For example, a block generator could require that transactions include AML/KYC information in its reference data. Since policy enforcement is not a part of the protocol rules, it is flexible, can be changed at will, and may use confidential information that should not be shared with the whole network.

## Future improvements

Future versions of the Chain Protocol may move further in the direction of full Byzantine-fault-tolerance, such as by supporting leader rotation or multi-phase commitment. Protocols such as [PBFT](http://pmg.csail.mit.edu/papers/osdi99.pdf) and [Tendermint](https://atrium.lib.uoguelph.ca/xmlui/bitstream/handle/10214/9769/Buchman_Ethan_201606_MAsc.pdf?sequence=7) show promise in this area. 

Future protocols may also include specifications of fraud proofs and gossip protocols to allow network participants to more easily detect and report problems in the network, such as forks or signatures on invalid blocks. 



