# Documentation

## Development

To view docs with their associated HTML, styles and fonts, we use a tool
called `md2html`.

Make sure all Chain Core commands have been installed by following the
installation instructions in the [repo README](../Readme.md#installation).

Once installed, run `md2html` from the root directory of the rep:

```sh
$ cd $CHAIN
$ go run ./cmd/md2html/*.go
```

The converted documentation is served at
[http://localhost:8080/docs](http://localhost:8080/docs).

---

## Table of Contents

### Chain Core

* Get Started
  * [Introduction](core/get-started/introduction.md)
  * [Install](index.html)
  * [SDKs](core/get-started/sdk.md)
  * [Configure](core/get-started/configure.md)
  * [5-Minute Guide](core/get-started/five-minute-guide.md)

* Build Applications
  * [Keys](core/build-applications/keys.md)
  * [Assets](core/build-applications/assets.md)
  * [Accounts](core/build-applications/accounts.md)
  * [Transactions](core/build-applications/transactions.md)
  * [Unspent Outputs](core/build-applications/unspent-outputs.md)
  * [Balances](core/build-applications/balances.md)
  * [Control Programs](core/build-applications/control-programs.md)
  * [Query Filters](core/build-applications/queries.md)
  * [Batch Operations](core/build-applications/batch-operations.md)

* Learn More
  * [Global vs Local Data](core/learn-more/global-vs-local-data.md)
  * [Blockchain Operators](core/learn-more/blockchain-operators.md)
  * [Blockchain Participants](core/learn-more/blockchain-participants.md)

* Reference
  * [API Objects](core/reference/api-objects.md)
  * [Product Roadmap](core/reference/product-roadmap.md)

### Chain Protocol

* Papers
  * [Whitepaper](protocol/papers/whitepaper.md)
  * [Federated Consensus](protocol/papers/federated-consensus.md)
  * [Blockchain Programs](protocol/papers/blockchain-programs.md)
  * [Blockchain Extensibility](protocol/papers/blockchain-extensibility.md)
  * [Protocol Roadmap](protocol/papers/protocol-roadmap.md)

* Specifications
  * [Data Model](protocol/specifications/data.md)
  * [Validation](protocol/specifications/validation.md)
  * [Consensus](protocol/specifications/consensus.md)
  * [Virtual Machine](protocol/specifications/vm1.md)
  * [ChainKD](protocol/specifications/chainkd.md)
