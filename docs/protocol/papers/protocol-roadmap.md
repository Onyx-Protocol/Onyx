# Chain Protocol Roadmap

The initial version of Chain Protocol is focused on providing a foundation for large-scale deployment and extensibility, so that sophisticated security and confidentiality features can be safely introduced in later releases. Updates to the protocol are intended to be deployed as [soft forks](blockchain-extensibility.md) that keep already deployed applications compatible with a new version of the protocol.

If you have feature requests or feedback, we want to hear from you. Join us on [GitHub](https://github.com/chain), [Slack](https://slack.chain.com), or the [Chain developer forum](https://support.chain.com).

**Note:** This roadmap focuses on protocol improvements and does not cover features of Chain Core, which are discussed separately in the [Product Roadmap](../../core/reference/product-roadmap.md).

## Denial of service mitigation

* Fine-tuned and tested runtime costs for the virtual machine
* Adjustable run limit, as well as adjustable limits on the size of blocks, transactions, programs, and other data structures
* Improved consensus algorithms to guarantee liveness under weaker assumptions

## Privacy

* Confidential transactions that hide assets and amounts through homomorphic encryption
* Untraceable transactions that further hide the identities of transacting parties

## Programs

* Simpler and more flexible virtual machine designs
* Support for arithmetic on homomorphically encrypted values
* Further development of high-level compiled languages
* Formal verification tools to evaluate program security

## Scalability

* Shifting memory and CPU burdens from the network to transacting parties
* More extensive support for compact proofs to protect light clients