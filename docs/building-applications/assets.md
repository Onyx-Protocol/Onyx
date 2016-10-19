# Assets

## Introduction

An asset is a set of fungible units that can be issued on a blockchain to represent any type of value. Once on a blockchain, units of an asset can be transacted directly between parties without the involvement of the issuer. The issuer can issue as many units as they want, as many times as they want.

Each asset has a globally unique asset ID that is derived from an issuance program. The issuance program defines a set of private keys and a quorum of signatures that must be provided to issue units of the assets. Chain Core automatically creates the issuance program when the asset is created.

Each asset can optionally include an asset definition, consisting of arbitrary key-value data. The asset definition is committed to the blockchain for all participants to see. Additionally, an asset can be tagged locally with private data for convenient queries and operations. For more information, see [Global vs. Local Data](../learn-more/global-vs-local-data).

## Overview

This guide will walk you through the basic functions of an asset:

* Create asset
* List assets (by asset definition, tags, and origin)
* Issue asset units to a local account (in the same Chain Core)
* Issue asset units to an external party
* Trade asset units by issuing to an external party
* Retire asset units
* List transactions (for issuance, transfer, and retirement)
* Get asset circulation

This guide assumes you know the basic functions presented in the [5-Minute Guide](../getting-started/five-minute-guide).

## Create asset

Creating an asset defines the asset object locally in the Chain Core. It does not exist on the blockchain until units are issued in a transaction.

* An `alias` is an optional, user-supplied, unique identifier that you can use to operate on the asset. We will use this later to build a transaction issuing units of the asset.
* The `quorum` is the threshold of keys that must sign a transaction to spend asset units controlled by the account.
* A `definition` is global data about the asset that is visible in the blockchain. We will create several fields in the definition.
* A `tag` is an optional key-value field used for arbitrary storage or queries. This data is local to the Chain Core and *not* visible in the blockchain. We will add several tags.

Create an asset for Acme Common stock:

$code ../examples/java/Assets.java create-asset-acme-common

Create an asset for Acme Preferred stock:

$code ../examples/java/Assets.java create-asset-acme-preferred

## List assets

Chain Core keeps a list of all assets in the blockchain, whether or not they were issued in the Chain Core. Each asset can be locally annotated with an alias and tags to enable efficient actions and intelligent queries. Note: local data is not present in the blockchain. For more information, see: [Local vs. Global Data](#).

To list all assets created in the Core, we build an assets query, filtering to the `origin` tag.

$code ../examples/java/Assets.java list-local-assets

To list all assets defined as preferred stock of a private security, we build an assets query, filtering to several tags.

$code ../examples/java/Assets.java list-private-preferred-securities

## Issue asset units to a local account

To issue units of an asset into an account within the Chain Core, we can build a transaction using an `asset_alias` and an `account_alias`.

We first build a transaction issuing 1000 units of Acme Common stock to the Acme treasury account.

$code ../examples/java/Assets.java build-issue

Once we have built the transaction, we need to sign it with the key used to create the Acme Common stock asset.

$code ../examples/java/Assets.java sign-issue

Once we have signed the transaction, we can submit it for inclusion in the blockchain.

$code ../examples/java/Assets.java submit-issue

## Issue asset units to an external party

If you wish to issue asset units to an external party, you must first request a control program from them. You can then build, sign, and submit a transaction issuing asset units to their control program.

We will issue 2000 units of Acme Common stock to an external party.

$code ../examples/java/Assets.java external-issue

### Retire asset units

To retire units of an asset from an account, we can build a transaction using an `account_alias` and `asset_alias`.

We first build a transaction retiring 50 units of Acme Common stock from Acme's treasury account.

$code ../examples/java/Assets.java build-retire

Once we have built the transaction, we need to sign it with the key used to create Acme's treasury account.

$code ../examples/java/Assets.java sign-retire

Once we have signed the transaction, we can submit it for inclusion in the blockchain.

$code ../examples/java/Assets.java submit-retire

## List asset transactions

Chain Core keeps a time-ordered list of all transactions in the blockchain. These transactions are locally annotated with asset aliases and asset tags to enable intelligent queries. Note: local data is not present in the blockchain. For more information, see: [Local vs. Global Data](#).

### Issuance transactions

To list transactions where Acme Common stock was issued, we build an assets query, filtering to inputs with the `issue` action and the Acme Common stock `asset_alias`.

$code ../examples/java/Assets.java list-issuances

### Transfer transactions

To list transactions where Acme Common stock was transferred, we build an assets query, filtering to inputs with the `spend` action and the Acme Common stock `asset_alias`.

$code ../examples/java/Assets.java list-transfers

### Retirement transactions

To list transactions where Acme Common stock was retired, we build an assets query, filtering to outputs with the `retire` action and the Acme Common stock `asset_alias`.

$code ../examples/java/Assets.java list-retirements

## Get asset circulation

The circulation of an asset is the sum of all asset units controlled by any control program (existing in unspent_outputs) in the blockchain.

To list the circulation of Acme Common stock, we build a balance query, filtering to the Acme Common stock `asset_alias`.

$code ../examples/java/Assets.java list-acme-common-balance

To list the circulation of all classes of Acme stock, we build a balance query, filtering to the `issuer` field in the `definition`.

$code ../examples/java/Assets.java list-acme-balance

To list all the control programs that hold a portion of the circulation of Acme Common stock, we build an unspent outputs query, filtering to the Acme Common stock `asset_alias`.

$code ../examples/java/Assets.java list-acme-common-unspents

[Download Code](../examples/java/Assets.java)
