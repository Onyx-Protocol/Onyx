# Balances

## Introduction

Any balance on the blockchain is simply a summation of unspent outputs. For example, the balance of Alice’s account is a summation of all the unspent outputs whose control program was created from the keys in Alice’s account.

Unlike other queries in Chain Core, balance queries do not return Chain Core objects, only simple sums over the `amount` fields in a specified list of unspent output objects.

### Sum By

Balance sums are totalled by `asset_id` and `account_id` by default, but it is also possible to query more complex sums. For example, if you have a network of counterparty-issued IOUs, you may wish to calculate the account balance of all IOUs from different counterparties that represent the same underlying currency.

## Overview

This guide will walk you through a few basic balance queries:

* [List account balances](#list-account-balances)
* [Get asset circulation](#get-asset-circulation)
* [List account balances, with custom summation](#list-account-balances-with-custom-summation)

### Sample Code

All code samples in this guide are extracted from a single Java file.

<a href="../examples/java/Balances.java" class="downloadBtn btn success" target="\_blank">View Sample Code</a>

## List account balances

List the asset IOU balances in Bank1's account:

$code ../examples/java/Balances.java account-balance

## Get asset circulation

Get the circulation of the Bank 1 USD IOU on the blockchain:

$code ../examples/java/Balances.java usd-iou-circulation

## List account balances with custom summation

List the asset IOU balances in Bank1's account, summed by currency:

$code ../examples/java/Balances.java account-balance-sum-by-currency
