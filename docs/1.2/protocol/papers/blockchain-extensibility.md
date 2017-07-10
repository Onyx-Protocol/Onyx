<!---
This document covers the extensibility features and versioning semantics of Chain Protocol that facilitate safe evolution of the protocol rules.
-->

# Blockchain Extensibility

* [Introduction](#introduction)
* [Hard forks and soft forks](#hard-forks-and-soft-forks)
* [Safety considerations](#safety-considerations)
* [Extensive and restrictive upgrades](#extensive-and-restrictive-upgrades)
* [Extension points](#extension-points)
* [Versioning rules](#versioning-rules)
* [Assets and programs interoperability](#assets-and-programs-interoperability)
* [Consensus upgrades](#consensus-upgrades)
* [Deprecation process](#deprecation-process)
* [Examples](#examples)
* [Conclusion](#conclusion)

## Introduction

Upgrading blockchain networks poses serious challenges. While traditional communications protocols only require point-to-point compatibility and allow the network as a whole to have varying degrees of interoperability, all nodes on a blockchain network must validate the same blocks. It is therefore impossible for any subset of parties to freely upgrade to an arbitrary new protocol without affecting the rest of the network (or, more likely, permanently detaching from it by “forking the chain”). As a result, for any protocol change to be implemented and deployed, the following issues must be considered:

* **Coordination:** how do all nodes agree on adoption of the new rules?
* **Transition security:** does transition to the new rules (even when done correctly) opens up security loopholes or breaks some assumptions?
* **Transition reliability:** is the transition implemented correctly and how likely it to contain bugs and cause the blockchain to fork?
* **Feature security:** is the upgraded protocol secure as advertised?

This document covers the extensibility features and versioning semantics of Chain Protocol that facilitate safe evolution of the protocol rules.

## Hard forks and soft forks

There are two ways to upgrade a blockchain network. These are conventionally known as *hard forks* and *soft forks*.

[sidenote]

See [BIP 99](https://github.com/bitcoin/bips/blob/master/bip-0099.mediawiki) for some background on the Bitcoin community's approach to hard and soft forks.

[/sidenote]

A **hard fork** is a change that is not compatible with non-upgraded software — it makes messages that were invalid under the old rules become valid under new rules. Changes to the protocol can be implemented in a straightforward manner, but at the cost of upgrading all participating nodes and all software that works directly with blockchain data structures. In a smaller network, a hard fork may be an efficient way to upgrade: one should set a deadline when the new rules will be activated, then update all necessary software before the deadline. Unfortunately, in larger networks hard forks pose a serious risk of breaking software unexpectedly, or would require a prohibitively distant deadline to give time to review and upgrade all software.

A **soft fork** is a compatible change — it rejects some messages that were previously valid. This may be a less elegant way to change the protocol, but it allows upgrading the network piece-by-piece without requiring every node to upgrade. A soft fork must be enforced by the block generation and consensus mechanism. For proof-of-work blockchains, this means the miners; for Chain blockchains, this means the block signers. A soft fork requires only block signers to upgrade first and enforce new rules by rejecting non-conforming transactions. To make sure future versions of the protocol still validate earlier blocks, block signers preemptively refuse to sign blocks containing transactions with undefined version numbers, making it safe to introduce new versions (and their associated new behavior) in the future. Once block signers begin enforcing new rules, other nodes can start creating transactions with a new version. Nodes that did not upgrade still validate these transactions, but implicitly rely on block signers to validate their exact meaning. Such nodes operate with reduced security regarding validation of new transactions. Nodes can upgrade to new capabilities at their own pace without blocking anyone else’s operation.

[sidenote]

Contrary to what one might assume, neither hard forks, nor soft forks are intended to _actually fork_ the blockchain. The word “fork” indicates the change of the protocol rules that _potentially_ may lead to an undesirable fork of the blockchain. While hard forks require careful coordination of all users, soft forks require careful coordination of block signers only.

[/sidenote]


## Safety considerations

### Safety of hard forks

While hard forks break compatibility with all fully validating nodes (running Chain Core or compatible software), some hard forks may be made compatible with a wider collection of software that verifies only a subset of protocol features. For instance, signing software deployed in HSMs, or clients that do not execute transaction programs, but only check transactions’ Merkle paths to a signed block.

A hard fork upgrade may have different compatibility implications, depending on its scope:

1. Changing transaction format: virtually no software remains compatible.
2. Changing block format: some transaction-signing software stays compatible.
3. Changing built-in limits: clients that verify compact proofs (transaction contents, Merkle paths to signed blocks) remain compatible.
4. Changing VM bytecode: some clients may remain compatible depending on character of the change.

In the best case incompatibility caused by hard forks causes outage for non-upgraded clients. But in general, it opens all clients (whether upgraded and not) to network forks and double-spending attacks that are hard to automatically detect and protect against since both sides of the network would see the alternative chain as completely invalid and consider it a noise. Hard fork upgrades are also harder to monitor: an unexpected incompatibility may remain hidden until it is too late and it causes the actual network fork.

Due to the difficulty of deploying hard forks safely, Chain Protocol provides versioning and upgrade semantics for blockchain data structures in order to enable safe soft fork upgrades.


### Safety of soft forks

A soft fork upgrade _adds_ rules and _does not remove_ existing ones. As a result, non-upgraded nodes see the upgraded blockchain as valid and easily detect a fork caused by block signers. Upgraded nodes can also detect a fork caused by block signers and pinpoint it to a violation of a specific rule. Although all soft forks provide basic safety guarantee in respect to forking the ledger, the actual safety depends on the nature and scale of the upgrade.

Soft forks allow implementing a surprisingly large number of changes, from bug fixes and small features in the VM, to introducing a completely new block structure. When the versioning rules are carefully planned in advance, changes to the software can also be implemented cleanly.

Smaller changes are safer to implement as they touch smaller software surface. An example of a relatively safe change is addition of a new instruction to the VM in place of one of the NOP instructions: non-upgraded users cannot use it by mistake and are able to execute the rest of the contract logic using pre-existing rules (so the surface of a possible attack is very narrow). It is much less safe to introduce a new transaction format: non-upgraded nodes will entirely ignore the new transactions, a larger amount of software will have to be upgraded to take advantage of the new format. Then even more complexity will have to be added to make old and new transaction formats interoperable.


## Extensive and restrictive upgrades

Soft forks allow two approaches to network upgrades:

1. **Extensive upgrade:** introduction of a new rule (security feature) the co-exists with the existing ones. A pre-existing extension field in the blockchain is extended with additional data having a specific meaning applied.
2. **Restrictive upgrade:** usage of existing features is restricted or partially limited by censoring transactions. Block signers can restrict some transactions anyway for performance or security reasons, but they can also commit to such behaviour by making it a protocol rule so that other nodes can rely on it.

The first approach allows introducing new features for security or performance reasons without disrupting the use of the existing features.

[sidenote]

Note that restricting existing feature and adding a new one is not equivalent to redefining the rules that apply to a certain piece of data. The former remains a soft fork, while the former is a hard fork. The non-upgraded nodes ignore the new feature and stop seeing instances of usage of the old feature and therefore stay compatible with the blockchain.

[/sidenote]

The second approach allows for performance optimization and deprecation of less secure features. If a part of the protocol is proven to be completely insecure, a “fail-stop” soft fork upgrade can immediately prohibit its usage.

Together, the first and the second approaches allow implementing a tick-tock upgrade schedule: first, rolling out a new security feature, then phasing out support for the deprecated one until all nodes are upgraded and the network can forbid its use as a protocol rule.



## Extension points

Chain Protocol defines four main areas of extensibility:

1. Blocks
2. Transactions
3. Assets
4. Programs

Every extension area includes several _extension points_ with specified upgrade semantics.


### Block extension points

Field          | Semantics
---------------|---------------------------------
Block Version  | Monotonically-increased integer. New value indicates to all nodes that a new protocol rule is being activated.
Block Commitment | Additional commitments are appended to the string and can be ignored by non-upgraded nodes.
Block Witness | Additional proofs are appended to the string and can be safely ignored by non-upgraded nodes.

Note that block version is the only version field that must be incremented monotonically, and it does not require increments by 1. All other version fields may have arbitrary values in no particular order of introduction. At any given time, the version of a data structure either belongs to a set of known versions, or is simply *unknown*.


### Transaction extension points

Field          | Semantics
---------------|---------------------------------
Transaction version | Arbitrary integer allowing nodes to opt-in additional rules applied to transaction data.
Common fields  | Additional transaction fields common to all inputs and outputs can be committed to the existing transaction by appending them to this field. Non-upgraded nodes ignore unknown suffixes.
Common witness | Additional witness data may be appended to the string to aid in validation of the blockchain. Non-upgraded nodes ignore unknown suffixes.


### Asset extension points

Field             | Semantics
------------------|---------------------------------
Asset Version     | Variable-length integer identifying an asset accounting scheme and capabilities. Inputs and outputs with unknown asset versions are ignored by the nodes.
Output Commitment | Additional “deposit” commitments are appended to the string and can be ignored by non-upgraded nodes.
Input Commitment  | Additional “spending” commitments are appended to the string and can be ignored by non-upgraded nodes.
Input Witness     | Additional witness data may be appended to the string in one or more asset versions to aid in validation of the blockchain. Non-upgraded nodes ignore unknown suffixes.


### Program extension points

Transaction programs have five extension points within the output and issuance programs.

Block programs are not versioned and can be upgraded only via additional block commitment elements.

Field         | Semantics
--------------|---------------------------------
VM Version    | Variable-length integer that identifies VM for the issuance and output programs. Nodes first read the version field and if the version is unknown, resolve the program without parsing it as if it was executed successfully. This way, arbitrary VM changes can be made via version updates.
NOP* instructions  | NOP instructions by default do nothing, but can be redefined to perform additional verification without effect on the stack, but behaving like CHECK*SOMECONDITION*VERIFY, without VM version updates.

**Note:** all unassigned opcodes are defined as NOPs (“no operation”) that enable minor changes to the VM without changing the VM version. So the non-upgraded nodes will be able to validate most of the program without having to ignore it completely as “anyone can spend” as a new VM version would cause them to do. Even when the VM version needs to be increased (e.g. a new instruction need a significantly larger run cost or does non-trivial changes to VM state), keeping existing instructions untouched allows keeping consistency between VM implementations and related tools and avoiding bugs.


## Versioning rules

Chain Protocol implements a versioning scheme that allows users signalling whether they opt into new version of the protocol and also enable the network to protect the extension points (more on them below) against abuse before they have specific semantics assigned to them.

All upgrades are sequential. Nodes cannot upgrade to a protocol extension without upgrading to all previous ones. Users that do not validate the blockchain fully may cherry-pick which features to validate themselves or delegate trust to block signers or third-party data providers. However, they still must be aware of all intermediate extensions when deciding to opt out of validating some aspects of the protocol. This is reflected in the requirement that block versions monotonically increase: protocol rules that were added once cannot be rolled back in the future. All other version fields while encoded as variable-length integers are free-form and may be treated as integers, short strings, bit fields or a combination thereof.

**Extensive upgrades** to the protocol are signaled by incrementing the block version and changing the versions of all affected data structures that introduce the additional rules. Use of unassigned fields and instructions is *not allowed* without changing the version of the data structure that is being extended and all data structures up in the hierarchy up to the block version. This allows applications to detect addition of new fields, asset and program versions to the protocol and adjust their security assumptions, notify administrators about an upgrade, or take other action as appropriate. This also provides security for users of compact proofs: they are guaranteed that they will notice a protocol upgrade when the previously unused fields become defined.

**Restrictive upgrades** — that limit use of defined fields and instructions — only need to increment a block version in order to maintain compatibility of the new rules with the historical blocks. For additional compatibility with transacting software the following scheme is recommended:

1. The restrictions apply only to a new transaction version (or versions).
2. Previous transaction version is either allowed indefinitely (therefore allowing users to opt-in the new rule), or announced as deprecated and can be restricted in the future (phased out) or immediately (fail-stop scenario).

### How it works

Lets say, a new VM version is introduced with an additional input commitment field necessary for a new VM feature.

1. Since the new VM version is used, the transaction version must be changed.
2. Transaction version is changed, therefore block version must be increased.
3. Block signers make sure only well-defined block versions are used and signed. Before the upgrade, block signers reject transactions that use unassigned extension points and version numbers.
4. When the block version is increased, the rest of the network receives a clear signal that the protocol rules has changed. Non-upgraded nodes may choose to stop processing payments until they upgrade, or apply additional checks and confirmations out of band.


## Assets and programs interoperability

VM versions can be updated independently of asset versions. New VM versions can access all previously defined asset versions but not future ones. Newer asset versions may use previously defined VMs and additional VMs introduced in the future (if explicitly defined for these asset versions).

When a new asset version is introduced, existing VMs may be nominally upgraded for all older asset versions so that programs controlling older assets may “see” newer assets which allows building trust-minimized bridges between old and new asset versions.

Users may keep using existing assets alongside the new ones without updating to a new VM, but simply signing multi-input/multi-output transactions for synchronous trades. They can also use an updated VM version with old assets in order to build contracts that involve both new and old asset version.

Issuers may continue to recognize older assets, offer a trusted exchange of old assets for the new ones or define new asset versions via issuance programs that perform conversion in a trust-minimized manner (e.g. a program may allow issuance of a new asset with the amount corresponding to an amount of the older asset explicitly destroyed in the same transaction).


## Consensus upgrades

Consensus protocol is mostly separated from the blockchain validation rules as it allows block signers to agree on a single tip block that they need to sign. Block header has an extensible “block commitment” field that allows addition of new rules supporting changes to the consensus protocol.

Extra care must be taken about upgrading the VM used for consensus programs. While control and issuance programs used in transactions have a VM version associated with them, blocks intentionally designed without explicit versioning. When non-upgraded nodes have to evaluate a program with an unknown VM version, they skip the evaluation and instead defer to block signers to decide if transaction is valid. In case of a block consensus program, there is no “higher authority” to defer to. If the new consensus protocol requires a new feature in the VM, block commitment and block witness fields must be extended with a new consensus program and its arguments to be evaluated *in addition* to the old VM and old consensus program.

[sidenote]

Block signers should not optimize performance changing old consensus program to `OP_1` (always succeeds) because it will immediately open non-upgraded nodes to a vulnerability: anyone would be able to fork the blockchain for them and make it look perfectly valid.

In some cases, block signers might want to change the old consensus program to `OP_FAIL` in case the old VM has an unavoidable security problem and it is safer to cause non-upgraded nodes to stop entirely and not accept any new blocks until they upgrade. Such update will become a _safe hard fork_ since the original chain will be prohibited from growing further.

[/sidenote]

While it is not easy to add support for new signature algorithms (e.g. changing an elliptic curve), VM 1 is flexible enough to allow implementing Lamport signatures: one-time hash-based signatures using built-in instructions for string manipulation, bitwise operations and hashing.


## Deprecation process

By default, all previously supported features remain supported to keep older software compatible. Sometimes features are discovered to be either insecure or inefficient and should be deprecated. Deprecation can be implemented as the block signers’ policy to refuse to include transactions using such features or restricting some aspects of them (size, frequency, etc). If necessary, deprecation can be implemented via a *restrictive upgrade* to the protocol enforced by the entire network. For instance, if a certain signature scheme is deemed insecure, it can be fully banned after a pre-arranged deadline to prevent inclusion of insecure transactions.


## Examples

### 1. Adding a new programming feature

We can add or modify an instruction, introduce a new signature scheme, or completely redesign the bytecode format by introducing a new VM version:

1. Specify an update to the format or semantics of output programs and corresponding signature programs.
2. Allocate a new VM version number to identify the new VM.
3. If necessary, specify additional data in the output commitment or input commitment (or both) for outputs using the new VM version.
4. If necessary, specify an additional global state commitment in the block commitment field.
5. Upgrade all block signers with software implementing the new feature.
6. At a pre-agreed time, increment the block version to signal the protocol update to all nodes.
7. Remaining nodes may upgrade and begin using the new VM.

### 2. Adding a new accounting scheme

We can implement a new asset accounting mechanism with new confidentiality or performance features by introducing a new asset version:

1. Specify a new protocol for issuance and transfer of assets, and a corresponding layout for input and output commitments and witness structures.
2. Allocate a new asset version to identify inputs and outputs using the new assets.
3. If necessary, specify an additional global state commitment in the block commitment field.
4. Upgrade all block signers with software implementing the new feature.
5. At a pre-agreed time, increment the block version to signal the protocol update to all nodes.
6. Remaining nodes may upgrade and begin using new asset versions.
7. Old assets can be phased out manually by having the issuer redeem them for new assets, or via a fully automated issuance program that does not require interaction with the issuer.

### 3. Updating core data structures

The two cases above demonstrated how some specific parts of the transaction structure can be updated. However, blocks are extensible enough to allow an entirely new transaction format introduced via a soft fork. For example, assume that the SHA3 hash function is found to be insecure and the network must migrate to a hypothetical “SHA4” algorithm:

1. The block commitment is extended with a previous block hash 2 field, defined as SHA4 of the previous block, as well as challenge program 2, that implements a new VM using SHA4.
2. The block witness is extended with signature program 2 that contains proofs matching challenge program 2.
3. A new asset version is introduced that uses SHA4 in asset IDs, programs, and other asset-specific data structures via input and output commitments.
4. The block commitment is extended again with additional state and transaction Merkle roots computed with SHA4. Previous commitments using SHA3 remain.
5. At a certain block height, the block version is incremented to signal the protocol update to all nodes.
6. Nodes update to support new asset versions and validate the new SHA4-based block commitments.


## Conclusion

The Chain Protocol includes a versioning scheme that facilitates network-wide upgrades. The scheme enables introduction of new features and improvements in a safe and compatible manner, providing meaningful security both for the network nodes and clients that rely on compact proofs instead of full blockchain validation.
