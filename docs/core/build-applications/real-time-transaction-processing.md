# Real-time Transaction Processing

## Overview

You can use **transaction feeds** to process transactions as they arrive on the blockchain. This is helpful for real-time applications such as notifications or live-updating interfaces. Transaction feeds work efficiently, without the need for polling or keeping state in your application.

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/RealTimeTransactionProcessing.java)
- [Ruby](../examples/ruby/real_time_transaction_processing.rb)

## Example

To illustrate how to use transaction feeds, we'll create a program that prints information about new [local](../learn-more/global-vs-local-data.md) transactions as they arrive.

#### Creating and retrieving a transaction feed

Transaction feeds can be created either in the Chain Core Dashboard, or from your application. When creating a feed, you can provide a [transaction filter expression](../build-applications/queries.md) so that the feed only contains transactions matching the filter. If you don't supply a filter, then the feed will contain all new transaction activity on the blockchain.

First, we'll create a new feed programmatically, setting the filter expression to `is_local='yes'`.

$code create-feed ../examples/java/RealTimeTransactionProcessing.java ../examples/ruby/real_time_transaction_processing.rb

From now on, we can retrieve this feed using its alias:

$code get-feed ../examples/java/RealTimeTransactionProcessing.java ../examples/ruby/real_time_transaction_processing.rb

The Chain Core will record how much of a feed has been processed, so your application doesn't have to keep track itself.

#### Handling incoming transactions from the feed

To process a transaction, our example application will print out some basic information to the console:

$code processor-method ../examples/java/RealTimeTransactionProcessing.java ../examples/ruby/real_time_transaction_processing.rb

Next, we'll set up an infinite loop that reads from the transaction feed, and sends each incoming transaction to our processor function:

$code processing-loop ../examples/java/RealTimeTransactionProcessing.java ../examples/ruby/real_time_transaction_processing.rb

Note the call to `ack` at the end of every cycle. This updates the transaction feed via an API call so that if your program terminates for any reason, it can pick back up from the point that `ack` was last called.

The body of the processing loop will run once for every new transaction that arrives on the blockchain. If you've already processed all available transactions, then the call to `next` will **block the active thread** until a transaction matching the filter arrives. Because of this blocking behavior, we'll run the processing loop in its own thread:

$code processing-thread ../examples/java/RealTimeTransactionProcessing.java ../examples/ruby/real_time_transaction_processing.rb

#### Testing the example

In order to push some transactions through the transaction feed, we'll try generating a sample transaction:

$code issue ../examples/java/RealTimeTransactionProcessing.java ../examples/ruby/real_time_transaction_processing.rb

Almost immediately, we should see the following output in the console:

```
New transaction at Sun Oct 16 17:08:53 PDT 2016
  ID: 7735ac967928ef04436dcaf836d22e17ab0d3fc72b769361e8506642e515ab65
  Input 0
    Type: issue
    Asset: gold
    Amount: 100
    Account: null
  Output 0
    Type: control
    Purpose: receive
    Asset: gold
    Amount: 100
    Account: alice
```

Let's try submitting another transaction:

$code transfer ../examples/java/RealTimeTransactionProcessing.java ../examples/ruby/real_time_transaction_processing.rb

This should result in the following output:

```
New transaction at Sun Oct 16 17:08:55 PDT 2016
  ID: a934af2bef5187ad17bf51c4631aba44339478495cad11900acd95d5e510d91c
  Input 0
    Type: spend
    Asset: gold
    Amount: 100
    Account: alice
  Output 0
    Type: control
    Purpose: change
    Asset: gold
    Amount: 50
    Account: alice
  Output 1
    Type: control
    Purpose: receive
    Asset: gold
    Amount: 50
    Account: bob
```

## Subtleties

#### Order of transactions

A transaction feed provides transactions in the order they are arranged on the blockchain, from earliest to most recent, starting with transactions that arrive on the blockchain immediately after the feed was created.

#### Efficiency

Under the hood, the SDK reads data from a transaction feed using a long-polling mechanism. This ensures that network round trips between your application and the Chain Core are kept to a minimum.

#### When to call `ack`

Calling `ack` for each cycle of your processing loop is the safest strategy, but it's not the only strategy. If you'd prefer to cut down on API calls to the Chain Core, you can call `ack` less frequently. The less frequently you call `ack`, the more risk you'll have of repeating some processing if your program terminates unexpectedly.

Transaction feeds provide *at-least-once* delivery of transactions; occasionally, a transaction may be delivered multiple times. Regardless of how frequently you call `ack`, it's a good idea to design your transaction processing to be **idempotent**, so that your application can process a given transaction twice or more without adverse effects.

#### Concurrency

As mentioned in the example, reading from a transaction feed may block your active process, so if your application does more than just consume a transaction feed, you should run the processing loop within its own thread.

In general, you should consume a transaction feed in one and only one thread. In particular, you'll want to make sure that `next` and `ack` are called serially, within a single thread.
