# Multiparty Trades

 This guide demonstrates how to use the Client API to create complex transactions that are:

- **Multi-party**: Multiple accounts on the same core or different cores can participate in the same transaction.
- **Multi-asset**: Multiple assets can be traded and issued in the same transaction. The assets may originate from any core.
- **Risk-free**: Within a single transaction, each party can declare exactly what they will pay, and what they expect to receive. The transaction will be rejected by the blockchain unless it is signed by all parties, and unless its incoming and outgoing assets balance each other.

Please make sure you've read [Transaction Basics](../building-applications/transactions) before continuing.

## Example: Trading within the same core and application

In this example, Alice and Bob each have accounts on the same core. Alice holds Alice Dollars, and Bob holds Bob Bucks. Their accounts are managed by a central application that has access to both of their HSMs.

In this setting, the application can trade Alice's Alice Dollars for Bob's Bob Bucks directly, within a single transaction:

$code ../examples/java/MultipartyTrades.java same-core-trade

This is the simplest possible scenario for a multi-asset trade between two accounts. Because the assets and accounts are local to the same core, the application can name all of the relevant accounts and assets in the space of a single `build` call. And since the application has access to both Alice and Bob's HSMs, it can sign for both parties simultaneously.

## Example: Trading between cores

Suppose that we want to conduct a similar trade as above, but Alice's account and Bob's account are now on different cores, and managed by separate applications.

Alice initiates the trade by building a transaction that stipulates what she will pay, and what she expects to receive. Since the Bob Buck asset is not local to her core, Alice can't refer to it by its human-readable alias. Instead, she has to know its asset ID from an out-of-band source, like an email from Bob.

$code ../examples/java/MultipartyTrades.java build-trade-alice

Unlike the transaction in the first example, this transaction is _unbalanced_. If Alice were to sign and submit the transaction at this point, it would be rejected by the blockchain.

However, Alice does have another option: she can provide a signature that commits her to her payment of Alice Dollars only if she receives Bob Bucks in the same transaction. She has to sign in such a way that the transaction can have more actions stacked on top of it.

To do this, Alice calls `allowAdditionalActions()` before she signs the transaction:

$code ../examples/java/MultipartyTrades.java sign-trade-alice

Now Alice as a signed, partial transaction that she can send to someone who can give her Bob Bucks in exchange for her Alice Dollars:

$code ../examples/java/MultipartyTrades.java base-transaction-alice

Alice takes the raw transaction, `baseTransactionFromAlice`, and emails it to Bob. Now Bob can fill in the rest of the trade, using the partially-signed transaction from Alice as a base transaction:

$code ../examples/java/MultipartyTrades.java build-trade-bob

As was the case with Alice, Bob can't refer to non-local assets by alias, so he refers to the Alice Dollar asset by its asset ID.

Note that the transaction now has all of the components of the trade described in the first example: it contains payments from Alice and Bob of their respective currencies, and names the other party as the recipients.

With Bob's addition, the transaction's incoming and outgoing assets are now balanced. But it's not a valid transaction without his signature. Since Bob is the last participant in the trade, he does _not_ call `allowAdditionalActions()` before he signs the transaction:

$code ../examples/java/MultipartyTrades.java sign-trade-bob

Finally, with the balanced transaction signed by both parties, Bob can submit the transaction to the blockchain network:

$code ../examples/java/MultipartyTrades.java submit-trade-bob

[Download Code](../examples/java/MultipartyTrades.java)
