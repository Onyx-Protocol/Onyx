# Balances

## Introduction

Any balance on the blockchain is simply a summation of unspent outputs. For example, the balance of Alice’s account is a summation of all the unspent outputs whose control program was created from the keys in Alice’s account.

Unlike other queries in Chain Core, balance queries do not return Chain Core objects, only simple sums over the `amount` fields in a specified list of unspent output objects.

### Sum By

Balance sums are totalled by `asset_id` and `asset_alias` by default, but it is also possible to query more complex sums. For example, if you have a network of counterparty-issued IOUs, you may wish to calculate the account balance of all IOUs from different counterparties that represent the same underlying currency.

## Overview

This guide will walk you through a few basic balance queries:

* [List account balances](#list-account-balances)
* [Get asset circulation](#get-asset-circulation)
* [List account balances, with custom summation](#list-account-balances-with-custom-summation)

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/Balances.java)
- [Ruby](../examples/ruby/balances.rb)

## List account balances

List the asset IOU balances in Bank1's account:

$code account-balance ../examples/java/Balances.java ../examples/ruby/balances.rb

## Get asset circulation

Get the circulation of the Bank 1 USD IOU on the blockchain:

$code usd-iou-circulation ../examples/java/Balances.java ../examples/ruby/balances.rb

## List account balances with custom summation

List the asset IOU balances in Bank1's account, summed by currency:

$code account-balance-sum-by-currency ../examples/java/Balances.java ../examples/ruby/balances.rb
