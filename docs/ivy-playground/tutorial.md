# Ivy Playground Tutorial

This tutorial shows you how to use the Ivy Playground, a new feature of Chain Core that lets you explore Ivy, Chain’s high-level contract language. **This is an early preview of Ivy provided for experimentation only. Not for production use.**

## Introduction

What is a blockchain contract?

You may already know that when Alice sends a payment to Bob using Chain Core, she “locks” the payment using Bob’s public key, and Bob (and only Bob) can “unlock” the payment in a later transaction by signing with the matching private key.

This lets every Chain Core node in the network answer the question, “Is this a valid spend of the payment Alice made?” They compare the digital signature in Bob’s transaction with the public key in Alice’s. If they match, it’s valid. If they don’t, it’s not.

But this is just one application of what is actually a much more general mechanism. Nodes in the network don’t ask “Does this signature match that public key?” Instead, they ask, “Does the _program_ in Alice’s transaction produce a true result when run with the _data_ in Bob’s?” Matching a signature against a public key is just one such program, but many others are possible.

The features and examples in the Ivy Playground demonstrate the kinds of things Ivy contracts can do.

## Before you get started

You’ll need to download and run a version of Chain Core Developer Edition that includes the experimental Ivy Playground feature. xxx

## Setting up

Open the Chain Core dashboard by visiting [http://localhost:1999/](http://localhost:1999/).

Select “Create new blockchain network” and click the “Create network” button. [xxx image]

Chain Core will ask if you’d like to try a 5-minute tutorial. [xxx image] This is not related to Ivy, but if you’re new to Chain Core, give it a try. It teaches concepts that will be useful in understanding Ivy. If you already have some familiarity with Chain Core, or you’re in a hurry, feel free to skip it.

Click “Ivy Playground” in the left-hand navigation bar. [xxx image]

## Welcome to the playground

The Ivy Playground opens on a view of the _contract template editor_, preloaded with the `LockWithPublicKey` example.

[xxx image]

`LockWithPublicKey` is an Ivy contract that performs the simple payment function described in the Introduction above: it is used to “lock” a payment, here called `value`, with a `PublicKey` supplied by the sender of the payment. A later transaction may `unlock` the `value` by supplying a `Signature` that passes the `checkTxSig` test. (That’s short for “check transaction signature.”)

You may edit the text of the contract, but first let’s look around at the other elements on the page.

Beneath the contract editor is a box labeled “Compiled.” It shows the _opcodes_ that Chain Core uses as a compact internal representation of the contract. (This is explained below.)

Above the contract editor are two buttons labeled “Load Template” and “Save.” The Save button lets you save contracts you create using the contract template editor. The Load Template button lets you load contracts you’ve previously saved, plus some prewritten samples we’ve provided, such as `LockWithPublicKey`. **Note** Contract templates are not saved permanently. They persist only as long as you leave the Ivy Playground open in your browser. [xxx is this right?]

Beneath the contract editor and “Compiled” box are two boxes, labeled “Value to Lock” and “Contract Arguments.” These are for using the selected contract to create a transaction that locks up actual value. We’ll return to this in a moment.

## Loading, editing, and saving contract templates

Click the “Load Template” button to get a list of available contracts. When you load a new one, the text of the contract template appears in the editor. Note how the “Compiled,” “Value to Lock,” and “Contract Arguments” sections also change.

Let’s experiment with editing a contract. But first, let’s talk a bit about the Ivy language.

### A bit about the Ivy language

A contract in Ivy always begins with the keyword `contract` followed by the contract’s name, the contract’s _parameters_, the keyword `locks`, and a name for the value controlled by the contract.

After this introduction comes one or more _clauses_. Each clause describes one possible way to unlock the value in the contract, and the data and/or payments needed to do so. In `LockWithPublicKey`, there is only one way to unlock the value: by providing a valid signature. But take a look at the `TradeOffer` contract (using the Load Template button). This contract locks up a piece of value called `offered` and there are two ways to unlock it:

- in `trade`, anyone who supplies the correct `payment` to the seller unlocks the value in `offered`
- in `cancel`, the seller supplies his or her signature to reclaim the value in `offered`

It’s worth revisiting the steps involved in using something like the `TradeOffer` contract. One transaction uses it to lock up some value:

- Alice has 100 euros and wants 110 US dollars
- She creates a program by setting the parameters of `TradeOffer` as follows:
  - `requestedAsset` is `USD` (US dollars)
  - `requestedAmount` is 110
  - `sellerProgram` is the program where Alice can receive payment [xxx digression about create receiver]
  - `sellerKey` is Alice’s public key
- Alice creates a transaction paying 100 euros from her account into the `TradeOffer` program she just created

A later transaction unlocks the value:

- Bob, who has dollars and wants euros, sees Alice’s transaction and creates a new one with two inputs and two outputs:
  - One input is Alice’s 100 euros, unlocked with the `trade` clause
  - One input is 110 dollars from Bob’s account
  - One output sends the 110 dollars to Alice’s `sellerProgram`
  - One output sends the 100 euros to Bob’s account

Or, alternatively, if Bob never comes along and Alice wants to withdraw her offer:

- Alice creates her own second transaction with:
  - One input that unlocks the 100 euros in the `TradeOffer` contract by supplying a signature matching the `sellerKey` specified earlier
  - One output paying those 100 euros back to Alice’s account

_Contract_ parameters (including the locked value) are specified as part of the earlier transaction. _Clause_ parameters (including any required payments) are specified as part of the later transaction.

### Loading, editing, and saving contract templates, continued

Using the Load Template button, load the `LockWithPublicKey` contract template. Let’s see what happens when we make different kinds of changes. **Note** At any point, if you can’t find your way back from an error, simply reload the page in your browser, or reload the `LockWithPublicKey` contract with the Load Template button, to start over.

Where the first line says `locks value`, change it to read `locks x`. You should see:

- the “Value to Lock” and “Contract Arguments” boxes disappear
- the “Compiled” box display the error, “value "x" not disposed in clause "spend"”

The error means that the contract locks some value (now named `x`) that is never unlocked anywhere.

Now, near the end of the `spend` clause, change `unlock value` to `unlock x`. Now that the value locked by the contract _is_ handled in `spend`, the error should disappear and the “Value to Lock” and “Contract Arguments” boxes should return.

Now try adding a parameter to `LockWithPublicKey`. Change the parameter list from `(publicKey: PublicKey)` to `(publicKey: PublicKey, deadline: Time)`. You should see the error, “parameter "deadline" is unused.” This illustrates that every contract parameter must be used at least once.

Return the `LockWithPublicKey` parameter list to `(publicKey: PublicKey)`. The error should go away. Now try adding an unused parameter to the `spend` clause, i.e. `clause spend(sig: Signature, deadline: Time)`. This too is an error.

Try other changes to see their effects. When you’re ready to continue, reload the page in your browser to start fresh.

Let’s try creating and saving a new contract template. It’ll be just like `LockWithPublicKey`, so Alice can use it to send payments to Bob, but it will have the added feature of a deadline. If the payment is unclaimed by the recipient after the deadline, it reverts to the sender. Let’s call this new contract `TimedPayment`.

1. Load the `LockWithPublicKey` template with the Load Template button
2. Change the beginning from `contract LockWithPublicKey` to `contract TimedPayment`
3. Add a `deadline` to the list of contract parameters [xxx image]
4. The `spend` clause should only be usable _before_ the deadline, so add the line `verify before(deadline)` to the `spend` clause [xxx image]

Now we need a new clause for unlocking `value` and returning it to the sender after the deadline passes. The naive way to do it is:

```
clause naiveRevert() {
    verify after(deadline)
    unlock value
}
```

This is naive because it places no constraint on where `value` is sent. _Anyone_ can unlock and claim `value` for themselves after the deadline.

Rather than allowing `value` to be sent anywhere, let’s instead require that it go back to the sender:

```
clause revert() {
    verify after(deadline)
    lock value with sender
}
```

If you insert this clause after the `spend` clause, the Ivy Playground should look like this: [xxx image]

There’s still an error: `sender` is undefined. [xxx the actual error message in this case is “program in lock statement in clause "revert" has type "", must be Program,” can we improve that?] It must be specified at the time the first transaction is created, i.e. as a contract parameter, so that the later transaction can’t send `value` anywhere it chooses. Add `sender: Program` to the contract parameters, and the contract template is complete: [xxx image]

Click the Save button to save the `TimedPayment` contract template.

## Locking value in a contract




Choose a contract template

Choose the value

Provide contract arguments

Submit and switch to dashboard to see the transaction

Spend Value

Choose a clause

Provide clause arguments

Submit and switch to dashboard to see the transaction
