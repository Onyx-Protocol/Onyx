# 5-Minute Guide

## Introduction

This guide will walk you through the basic functions of Chain Core:

* Initialize the SDK
* Create keys (in the Chain Core MockHSM)
* Initialize the HSM Signer
* Create an account
* Create an asset
* Issue asset units into an account
* Spend asset units from one account to another
* Retire asset units from an account

## Initialize the SDK

Create an instance of the SDK. By default, the SDK will try to access a core located at `http://localhost:1999`, which is the default if you're running Chain Core locally.

$code ../examples/java/FiveMinuteGuide.java create-client

## Create Keys

Create a new key in the MockHSM.

$code ../examples/java/FiveMinuteGuide.java create-key

## Initialize the HSM Signer

To be able to sign transactions, load the key into the HSM Signer, which will communicate with the MockHSM.

$code ../examples/java/FiveMinuteGuide.java signer-add-key

## Create an Asset

Create a new asset, providing an alias, key, and quorum. The quorum is the threshold of keys that must sign a transaction issuing units of the asset.

$code ../examples/java/FiveMinuteGuide.java create-asset

## Create an Account

Create an account, providing an alias, key, and quorum. The quorum is the threshold of keys that must sign a transaction to spend asset units controlled by the account.

$code ../examples/java/FiveMinuteGuide.java create-account-alice

Create a second account to interact with the first account.

$code ../examples/java/FiveMinuteGuide.java create-account-bob

## Issue Asset Units

Build, sign, and submit a transaction that issues new units of the `gold` asset into the `alice` account.

$code ../examples/java/FiveMinuteGuide.java issue

## Spend Asset Units

Build, sign, and submit a transaction that spends units of the `gold` asset from the `alice` account to the `bob` account.

$code ../examples/java/FiveMinuteGuide.java spend

## Retire Asset Units

Build, sign, and submit a transaction that retires units of the `gold` asset from the `bob` account.

$code ../examples/java/FiveMinuteGuide.java retire

[Download Code](../examples/java/FiveMinuteGuide.java)
