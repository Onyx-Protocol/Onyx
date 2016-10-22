# Chain Protocol Roadmap

The initial version of Chain Protocol is focused on providing a foundation for large-scale deployment and extensibility, so that sophisticated security and confidentiality features can be safely introduced in later releases. Updates to the protocol are intended to be deployed as [soft forks](blockchain-extensibility.md) that keep already deployed applications compatible with a new version of the protocol.

If you have feature requests or feedback, we want to hear from you. Join us on [GitHub](https://github.com/chain), [Slack](https://slack.chain.com), or the [Chain developer forum](https://support.chain.com).

**Note:** This roadmap focuses on protocol improvements and does not cover features of Chain Core, which are discussed separately in the [Product Roadmap](../../core/reference/product-roadmap.md).

## Denial of service mitigation

* Explicit limits on number and size of blockchain entities (size of the blocks, number of transactions etc).
* Fine-tuned runtime cost limits for the control and issuance programs.
* Improvements to consensus algorithm.

## Privacy

* Homomorphically encrypted asset identifiers and amounts to provide secrecy for balances and financial parameters of the transactions.
* Controlled traceability of the transactions; hiding the link between transaction inputs and the previous transactions’ outputs.

## Programs

* Generalizing virtual machine for on-chain and off-chain predicate evaluation.
* Support for arithmetic on homomorphically encrypted values (to improve confidentiality of on-chain programs).
* High-level programming language and formal verification toolkit.

## Scalability

* Reducing the amount of data to be stored by the nodes by requiring clients use more sophisticated proofs.
* More elaborate support for compact proofs to improve security of clients that do not validate the blockchain entirely.
* Additional support for merging blockchains.
