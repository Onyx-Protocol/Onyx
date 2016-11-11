# 5-Minute Guide

## Overview

This guide will walk you through the basic functions of Chain Core:

* [Initialize the SDK](#initialize-the-sdk)
* [Create keys](#create-keys) (in the Chain Core Mock HSM)
* [Initialize the HSM Signer](#initialize-the-hsm-signer)
* [Create an asset](#create-an-asset)
* [Create an account](#create-an-account)
* [Issue asset units into an account](#issue-asset-units)
* [Spend asset units from one account to another](#spend-asset-units)
* [Retire asset units from an account](#retire-asset-units)

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/FiveMinuteGuide.java)
- [Ruby](../examples/ruby/five_minute_guide.rb)

## Initialize the SDK

Create an instance of the SDK. By default, the SDK will try to access a core located at `http://localhost:1999`, which is the default if you're running Chain Core locally.

$code ../examples/java/FiveMinuteGuide.java create-client

## Create Keys

Create a new key in the Mock HSM.

$code ../examples/java/FiveMinuteGuide.java create-key

## Initialize the HSM Signer

To be able to sign transactions, load the key into the HSM Signer, which will communicate with the Mock HSM.

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
