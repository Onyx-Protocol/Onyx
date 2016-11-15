# Assets

## Introduction

An asset is a type of value that can be issued on a blockchain. All units of a given asset are fungible.

Units of an asset can be transacted directly between parties without the involvement of the issuer.

Each asset has a globally unique asset ID that is derived from an issuance program. The issuance program typically defines a set of possible signing keys and a threshold number of signatures that must be provided to authorize issuance of new units of the asset. Chain Core automatically creates the issuance program when the asset is created. The issuer can issue as many units as they want, as many times as they want. Custom issuance programs are possible that enforce further limits on when, whether, and by whom new units may be issued.

Each asset can optionally include an asset definition consisting of arbitrary key-value data. The asset definition is committed to the blockchain for all participants to see. Additionally, an asset can be tagged locally with private data for convenient queries and operations. For more information, see [Global vs. Local Data](../learn-more/global-vs-local-data.md).

## Overview

This guide will walk you through the basic functions of an asset:

* [Create asset](#create-asset)
* [List assets](#list-assets) (by asset definition, tags, and origin)
* [Issue asset units to a local account](#issue-asset-units-to-a-local-account) (in the same Chain Core)
* [Issue asset units to an external party](#issue-asset-units-to-an-external-party)
* [Retire asset units](#retire-asset-units)
* [List transactions](#list-asset-transactions) (for issuance, transfer, and retirement)
* [Get asset circulation](#get-asset-circulation)

This guide assumes you know the basic functions presented in the [5-Minute Guide](../get-started/five-minute-guide.md).

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/Assets.java)
- [Ruby](../examples/ruby/assets.rb)

## Create asset

Creating an asset defines the asset object locally in the Chain Core. It does not exist on the blockchain until units are issued in a transaction.

* The `alias` is an optional, user-supplied, unique identifier that you can use to operate on the asset. We will use this later to build a transaction issuing units of the asset.
* The `quorum` is the threshold number of the possible signing keys that must sign a transaction to issue units of this asset.
* The `definition` is global data about the asset that is visible in the blockchain. We will create several fields in the definition.
* The `tag` is an optional key-value field used for arbitrary storage or queries. This data is local to the Chain Core and *not* visible in the blockchain. We will add several tags.

Create an asset for Acme Common stock:

$code create-asset-acme-common ../examples/java/Assets.java ../examples/ruby/assets.rb

Create an asset for Acme Preferred stock:

$code create-asset-acme-preferred ../examples/java/Assets.java ../examples/ruby/assets.rb

## List assets

Chain Core keeps a list of all assets in the blockchain, whether or not they were issued by the local Chain Core. Each asset can be locally annotated with an alias and tags to enable efficient actions and intelligent queries. Note: local data is not present in the blockchain, see: [Global vs Local Data](../learn-more/global-vs-local-data.md).

To list all assets created in the local Core, we build an assets query, filtering on the `is_local` tag.

$code list-local-assets ../examples/java/Assets.java ../examples/ruby/assets.rb

To list all assets defined as preferred stock of a private security, we build an assets query, filtering on several tags.

$code list-private-preferred-securities ../examples/java/Assets.java ../examples/ruby/assets.rb

## Issue asset units to a local account

To issue units of an asset into an account within the Chain Core, we can build a transaction using an `asset_alias` and an `account_alias`.

We first build a transaction issuing 1000 units of Acme Common stock to the Acme treasury account.

$code build-issue ../examples/java/Assets.java ../examples/ruby/assets.rb

Once we have built the transaction, we need to sign it with the key used to create the Acme Common stock asset.

$code sign-issue ../examples/java/Assets.java ../examples/ruby/assets.rb

Once we have signed the transaction, we can submit it for inclusion in the blockchain.

$code submit-issue ../examples/java/Assets.java ../examples/ruby/assets.rb

## Issue asset units to an external party

If you wish to issue asset units to an external party, you must first request a control program from them. You can then build, sign, and submit a transaction issuing asset units to their control program.

We will issue 2000 units of Acme Common stock to an external party.

$code external-issue ../examples/java/Assets.java ../examples/ruby/assets.rb

## Retire asset units

To retire units of an asset from an account, we can build a transaction using an `account_alias` and `asset_alias`.

We first build a transaction retiring 50 units of Acme Common stock from Acme’s treasury account.

$code build-retire ../examples/java/Assets.java ../examples/ruby/assets.rb

Once we have built the transaction, we need to sign it with the key used to create Acme’s treasury account.

$code sign-retire ../examples/java/Assets.java ../examples/ruby/assets.rb

Once we have signed the transaction, we can submit it for inclusion in the blockchain. The asset units in this transaction become permanently unavailable for further spending.

$code submit-retire ../examples/java/Assets.java ../examples/ruby/assets.rb

## List asset transactions

Chain Core keeps a time-ordered list of all transactions in the blockchain. These transactions are locally annotated with asset aliases and asset tags to enable intelligent queries. Note: local data is not present in the blockchain, see: [Global vs Local Data](../learn-more/global-vs-local-data.md).

### Issuance transactions

To list transactions where Acme Common stock was issued, we build an assets query, filtering on inputs with the `issue` action and the Acme Common stock `asset_alias`.

$code list-issuances ../examples/java/Assets.java ../examples/ruby/assets.rb

### Transfer transactions

To list transactions where Acme Common stock was transferred, we build an assets query, filtering on inputs with the `spend` action and the Acme Common stock `asset_alias`.

$code list-transfers ../examples/java/Assets.java ../examples/ruby/assets.rb

### Retirement transactions

To list transactions where Acme Common stock was retired, we build an assets query, filtering on outputs with the `retire` action and the Acme Common stock `asset_alias`.

$code list-retirements ../examples/java/Assets.java ../examples/ruby/assets.rb

## Get asset circulation

The circulation of an asset is the sum of all non-retired units of that asset existing in unspent transaction outputs in the blockchain, regardless of control program.

To list the circulation of Acme Common stock, we build a balance query, filtering on the Acme Common stock `asset_alias`.

$code list-acme-common-balance ../examples/java/Assets.java ../examples/ruby/assets.rb

To list the circulation of all classes of Acme stock, we build a balance query, filtering on the `issuer` field in the `definition`.

$code list-acme-balance ../examples/java/Assets.java ../examples/ruby/assets.rb

To list all the control programs that hold a portion of the circulation of Acme Common stock, we build an unspent outputs query, filtering on the Acme Common stock `asset_alias`.

$code list-acme-common-unspents ../examples/java/Assets.java ../examples/ruby/assets.rb
