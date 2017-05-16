# Ivy Playground Tutorial

## Introduction

This tutorial shows you how to use the Ivy Playground, a new feature of Chain Core that allows you to explore Ivy, Chain’s high-level contract language.

You may already know that when Alice sends a payment to Bob using Chain Core, she “locks” the payment using Bob’s public key, and Bob (and only Bob) can “unlock” the payment in a later transaction by signing with the matching private key.

This allows every node in the blockchain network to answer the question, “Is this a valid spend of the payment Alice made?” They compare the digital signature in Bob’s transaction with the public key in Alice’s. If they match, it’s valid. If they don’t, it’s not.

But this is just one application of what is actually a much more general mechanism. Nodes in the network don’t ask “Does this signature match that public key?” Instead, they ask, “Does the program in Alice’s transaction produce a true result when run with the arguments in Bob’s?” The program can say “match a signature against this public key,” but it doesn’t have to; there are plenty of other things it can say instead.



Introduction - minimum needed if didn't read blog post
Download and Setup
get Chain Core DE w/ Ivy Playground
open dashboard and playground
seed accounts and assets
Write a contract template - step by step with Trade Offer
Lock Value
Choose a contract template
Choose the value
Provide contract arguments
Submit and switch to dashboard to see the transaction
Spend Value
Choose a clause
Provide clause arguments
Submit and switch to dashboard to see the transaction
