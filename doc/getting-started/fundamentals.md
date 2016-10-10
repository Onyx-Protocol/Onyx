# Chain Core Fundamentals

## Introduction
Chain Core is enterprise-grade software used to operate or participate in a blockchain. A blockchain, also known as a blockchain, is a new type of financial database that makes database records operate like transferable financial instruments. We refer to these records as digital assets. Unlike a traditional financial database, where one entity is responsible for updating the balances for all participants, a blockchain gives direct control of digital assets to the participants themselves. Each participant maintains a set of private keys that secures their digital assets on a blockchain. When they wish to transfer digital assets to another participant, they must create a ledger update, known as a transaction, and sign it with their private keys. Once the signed transaction is submitted to the blockchain, the ownership of the digital assets is immediately and permanently updated for both participants.

Similarly, the initial issuance of digital assets onto a blockchain is controlled by cryptographically signed transactions. Each issuer maintains a set of private keys that are required to issue a specific type of digital asset. A digital asset can represent any unit of value that is defined by the issuer. From governments currencies, to corporate bonds, to loyalty points, to IOUs, to internal deposits, Chain Core issues all digital assets into a common, interoperable format.

Whether the participants on a blockchain are individuals, companies, or systems, and whether asset transfers span countries, companies, or even divisions within a single company, a blockchain offers a secure, trustless way to issue, manage and transfer financial assets.

## Keys
Cryptographic private keys are the primary authorization mechanism on a blockchain. They control both the issuance and transfer of asset units. In a production environment, private keys are generated within an HSM (hardware security module) and their corresponding public keys are exported for use within Chain Core. In order to issue or transfer asset units on a blockchain, a transaction is created in Chain Core and sent to the HSM for signing. The HSM signs the transaction without ever revealing the private key. Once signed, the transaction can be successfully submitted to the blockchain.

For development environments, Chain Core provides a convenient MockHSM. The MockHSM API is identical to the HSM API, providing a seamless transition from development to production.

## Assets
An asset ID represents a globally unique set of fungible asset units that can be issued onto a blockchain. Each asset ID is derived from an issuance program that defines a set of private keys and a quorum of signatures that must be provided to issue asset units. You can issue as many units as you want, as many times as you want.

## Transactions
A blockchain consists of an immutable set of cryptographically linked transactions. Each transaction consists of one or more inputs and outputs. An input defines a source of asset units - either a new issuance of an asset ID, or existing asset units controlled by a control program in a previous transaction. An output defines an amount of asset units from the inputs to be controlled by a new control program.

## Control Programs
Control programs define the ownership of asset units on a blockchain. Each control program consists of a set of conditions that must be satisfied in order to release asset units from the control program. The simplest type of control program - an account control program -  defines a set of private keys and a quorum of signatures that must be provided to release the asset units. A special type of control program - a retirement control program - allows existing asset units to be retired from a blockchain and never spent again.

## Accounts
An account is a convenience object in Chain Core (not on the blockchain) that manages a set of related control programs. Each account tracks its control programs on the blockchain to calculate asset balances and annotate transaction history.
