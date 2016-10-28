# Draft: Chain Core API Spec

This document serves as the canonical source of the HTTP interface to Chain Core.
As the API crystallizes, we will add more thorough descriptions of behaviour and data requirements.

## Table of Contents

* [Errors](#errors)
  * [Error Object](#error-object)
* [MockHSM](#mockhsm)
  * [Key Object](#key-object)
  * [Create Key](#create-key)
  * [List Keys](#list-keys)
  * [Sign Transaction](#sign-transaction)
* [Assets](#assets)
  * [Asset Object](#asset-object)
  * [Create Asset](#create-asset)
  * [List Assets](#list-assets)
* [Accounts](#accounts)
  * [Account Object](#account-object)
  * [Create Account](#create-account)
  * [List Accounts](#list-accounts)
* [Control Programs](#control-programs)
  * [Create Control Program](#create-control-program)
* [Transactions](#transactions)
  * [Transaction Object](#transaction-object)
  * [Unspent Output Object](#unspent-output-object)
  * [Transaction Template Object](#transaction-template-object)
  * [Build Transaction](#build-transaction)
  * [Submit Transaction](#submit-transaction)
  * [List Transactions](#list-transactions)
  * [List Balances](#list-balances)
  * [List Unspent Outputs](#list-unspent-outputs)
* [Transaction Feeds](#transaction-feeds)
  * [Transaction Feed Object](#transaction-feed-object)
  * [Create Transaction Feed](#create-transaction-feed)
  * [Get Transaction Feed](#get-transaction-feed)
  * [List Transaction Feeds](#list-transaction-feeds)
  * [Update Transaction Feed](#update-transaction-feed)
  * [Delete Transaction Feed](#delete-transaction-feed)
* [Access Tokens](#access-tokens)
  * [Create Access Token](#create-access-token)
  * [List Access Tokens](#list-access-tokens)
  * [Delete Access Token](#delete-access-token)
* [Core](#core)
  * [Configure](#configure)
  * [Update Configuration](#update-configuration)
  * [Info](#info)
  * [Reset](#reset)

## Errors

### Error Object

```
{
  "code": <string>,
  "message": <string>,
  "detail": <string>,
  "temporary": <boolean>
}
```

## MockHSM

### Key Object

```
{
  "alias": "...",
  "xpub": "xpub1.."
}
```

### Create Key

#### Endpoint

```
POST /mockhsm/create-key
```

#### Request

```
{
  "alias": "..." // optional
}
```

#### Response

A [Key Object](#key-object).

### List Keys

#### Endpoint

```
POST /mockhsm/list-keys
```

#### Request

```
{
  "aliases": [<string>, ...], // optional
  "after": "..." // optional
}
```

#### Response

```
{
  "items": [
    <key object>,
    ...
  ],
  "next": {
    "aliases": [<string>, ...],
    "after": "..."
  },
  "last_page": true|false
}
```

### Sign Transaction

#### Endpoint

```
POST /mockhsm/sign-transaction
```

#### Request

```
{
  "transactions": [
    {
      <all fields in the Transaction Template object>,
      "allow_additional_actions": <boolean> // optional, defaults to false
    },
    ...
  ],
  "xpubs": ["..."]
}
```

#### Response

An array of [transaction template objects](#transaction-template-object) and/or [error objects](#error-object).

## Assets

### Asset Object

```
{
  "id": "...",
  "alias": "...",
  "issuance_program: "...",
  "keys": [
    {
      "root_xpub": "...", // only available if is_local is yes
      "asset_pubkey": "...",
      "asset_derivation_path": "..."  // only available if is_local is yes
    },
    ...
  ],
  "quorum": 1,
  "definition": {},
  "tags": {},
  "is_local": <"yes"|"no">
}
```

### Create Asset

#### Endpoint

```
POST /create-asset
```

#### Request

```
[
  {
    "alias": "...",
    "root_xpubs": ["..."],
    "quorum": 1,
    "definition: {},
    "tags": {}
  }
]
```

#### Response

An array of [asset objects](#asset-object).

### List Assets

#### Endpoint

```
POST /list-assets
```

#### Request

```
{
  "filter": "...",
  "filter_params": [], // optional
  "after": "..." // optional
}
```

#### Response

```
{
  "items": [
    <asset object>,
    ...
  ],
  "next": {
    "filter": "...",
    "filter_params": [],
    "after": "..."
  },
  "last_page": true|false
}
```

## Accounts

### Account Object

```
{
  "id": "...",
  "alias": "...",
  "keys": [
    {
      "root_xpub": "...",
      "account_xpub": "...",
      "account_derivation_path": "..."
    },
    ...
  ],
  "quorum": 1,
  "tags": {}
}
```

### Create Account

#### Endpoint

```
POST /create-account
```

#### Request

```
[
  {
    "alias": "...",
    "root_xpubs": ["xpub"],
    "quorum": 1,
    "tags": {}
  }
]
```

#### Response

An array of [account objects](#account-object).

### List Accounts

#### Endpoint

```
POST /list-accounts
```

#### Request

```
{
  "filter": "...", // optional
  "filter_params": [], // optional
  "after": "..." // optional
}
```

#### Response

```
{
  "items": [
    <account object>,
    ...
  ],
  "next": {
    "filter": "...",
    "filter_params": [],
    "after": "..."
  },
  "last_page": true|false
}
```

## Control Programs

### Create Control Program

#### Endpoint

```
POST /create-control-program
```

#### Request

```
[
  {
    "type": "...",
    "params": {}
  }
]
```

If the `type` is `account` then the following params are required:

```
{
  "account_id": "..." // accepts `account_id` or `account_alias`
}
```

#### Response

```
[
  {
    "control_program": "..."
  }
]
```

## Transactions

### Transaction Object

Annotated by the Core services where possible (account_ids, account_tags, asset_tags)

```
{
  "id": "C5D3F8...",
  "timestamp": "2015-12-30T00:02:23Z",
  "block_id": "A83585...",
  "block_height": 100,
  "position": ..., // position in block
  "reference_data": {"deal_id": "..."},
  "is_local": <"yes"|"no">, // local if any input or output is local
  "inputs": [
    {
      "type": "issue",
      "asset_id": "125B4E...",
      "asset_alias": "...",
      "asset_definition": <object>,
      "asset_tags": {},
      "asset_is_local": <"yes"|"no">
      "amount": 5000,
      "issuance_program": ...,
      "reference_data": {"details": "..."},
      "is_local": <"yes"|"no"> // local if type is issue and asset is local
    },
    {
      "type": "spend",
      "asset_id": "125B4E...",
      "asset_alias": "...",
      "asset_definition": <object>,
      "asset_tags": {},
      "asset_is_local": <"yes"|"no">,
      "amount": 5000,
      "spent_output": {
        "transaction_id": "94C5D3...",
        "position": 1,
      },
      "account_id": "",
      "account_alias": "...",
      "account_tags": {},
      "reference_data": {"user": "alice"},
      "is_local": <"yes"|"no"> // local if account id is not null
    }
  ],
  "outputs": [
    {
      "type": "control",
      "purpose": <"change"|"receive">, // provided if the control program was generated locally
      "position": "...",
      "asset_id": "125B4E...",
      "asset_alias": "...",
      "asset_definition": <object>,
      "asset_tags": {},
      "asset_is_local": <"yes"|"no">,
      "amount": 5000,
      "account_id": "",
      "account_alias": "...",
      "account_tags": {},
      "control_program": "205CDF...",
      "reference_data": {"user": "bob"},
      "is_local": <"yes"|"no"> // local if type is control and account id is not null
    }
  ]
}
```

Note: the "retire" SDK method is simply a control program containing the `RETURN` operation.
To keep the interface narrow, the SDK can generate such a control program.

### Transaction Template Object

```
{
  "raw_transaction": <hex string>,
  "signing_instructions": [
    {
      "position": 0,
      "asset_id": "2ed22e7846968aaee500b5ea4b4dfc8bdbe798f32e0737516ab44be4417ff111",
      "amount": 4,
      "witness_components": [
        {
          "type": "data",
          "data": "abcd..."
        },
        {
          "type": "signature",
          "quorum": <int>,
          "keys": [
            {
              "xpub": <string>,
              "derivation_path": [<int>, ...]
            }
          ],
          "program": <string>,
          "signatures": [<string>, ...]
        }
      ]
    }
  ]
}
```

### Unspent Output Object

```
{
  "type": "control",
  "purpose": <"change"|"receive">, // provided if the control program was generated locally
  "transaction_id": "...",
  "position": "...",
  "asset_id": "...",
  "asset_alias": "...",
  "asset_definition": <object>,
  "asset_tags": {},
  "asset_is_local": <"yes"|"no">,
  "amount": 5000,
  "account_id": "...",
  "account_alias": "...",
  "account_tags": {},
  "control_program": "...",
  "reference_data": {}
}
```

### Build Transaction

#### Endpoint

```
POST /build-transaction
```

#### Request

```
[
  {
    "base_transaction": <hex string>, // optional. an unsubmitted transaction to which additional actions can be appended.
    "actions": [
      {
        "type": "spend_account",
        "asset_id": "...", // accepts `asset_id` or `asset_alias`
        "amount": 123,
        "account_id": "...",
        "ttl": <number of milliseconds>, // optional, defaults to 300000 (5 minutes)
      },
      {
        "type": "spend_account_unspent_output",
        "transaction_id": "...",
        "position": 0,
        "reference_data": "...",
        "ttl": <number of milliseconds>, // optional, defaults to 300000 (5 minutes)
      },
      {
        "type": "issue",
        "asset_id": "...", // accepts `asset_id` or `asset_alias`
        "amount": 500,
        "reference_data": "...",
        "ttl": <number of milliseconds>, // optional, defaults to 300000 (5 minutes)
      },
      {
        "type": "control_account",
        "asset_id": "...", // accepts `asset_id` or `asset_alias`
        "amount": 500,
        "account_id": "...", // accepts `account_id` or `account_alias`
        "reference_data": "..."
      },
      {
        "type": "control_program",
        "asset_id": "...", // accepts `asset_id` or `asset_alias`
        "amount": 500,
        "control_program": "...",
        "reference_data": "..."
      },
      {
        "type": "set_transaction_reference_data",
        "reference_data": <object>
      }
    ]
  }
]
```

#### Response

An array of [transaction template objects](#transaction-template-object) and/or [error objects](#error-object).

### Submit Transaction

#### Endpoint

```
POST /submit-transaction
```

#### Request

```
[
  {
    "raw_transaction": <hex string>
  }
]
```

#### Response

```
[
  // Object with ID if transaction submission succeeded.
  {
    "id": "..."
  },

  // Error object if transaction submission failed.
  <error object>,

  ...
]
```

### List Transactions

#### Endpoint

```
POST /list-transactions
```

#### Request

```
{
  "filter": "...", // optional
  "filter_params": [], // optional
  "start_time": <number, millisecond Unixtime>, // optional, defaults to 0
  "end_time": <number, millisecond Unixtime>, // optional, defaults to current time
  "ascending_with_long_poll": <boolean>, // optional, defaults to false (newest to oldest, does not long poll)
  "after": "...", // optional
  "timeout": <number, in milliseconds> // optional, defaults to 1000 (1 second)
}
```

#### Response

```
{
  "items": [
    <transaction object>,
    ...
  ],
  "next": {
    "filter": "...",
    "filter_params": [],
    "start_time": <number>,
    "end_time": <number>,
    "ascending_with_long_poll": <boolean>,
    "after": "..."
  },
  "last_page": true|false
}
```

### List Balances

#### Endpoint

```
POST /list-balances
```

#### Request

```
{
  "filter": "...", // optional
  "filter_params": ["param"], // optional
  "sum_by": ["selector1", ...], // optional
  "timestamp": <number, millisecond Unixtime> // optional, defaults to current time
}
```

#### Response

Grouped:

```
{
  "items": [
    {
      "sum_by": {
        "selector1": "...",
        ...
      },
      "amount": ...
    },
    ...
  ],
  "next": {...},
  "last_page": true // currently only returns one page
}
```

Ungrouped:

```
{
  "items": [
    {
      "amount": 10
    }
  ],
  "next": {
    "filter": "...",
    "filter_params": [],
    "sum_by": [...],
    "timestamp": <number>,
    "after": "..."
  },
  "last_page": true
}
```

### List Unspent Outputs

#### Endpoint

```
POST /list-unspent-outputs
```

#### Request

```
{
  "filter": "...", // optional
  "filter_params": [], // optional
  "timestamp": <number, millisecond Unixtime>, // optional, defaults to current time
  "after": "..." // optional
}
```

#### Response

```
{
  "items": [
    <unspent output object>,
    ...
  ],
  "next": {
    "filter": "...",
    "filter_params": [],
    "timestamp": <number>,
    "after": "..."
  },
  "last_page": true|false
}
```

## Transaction Feeds

### Transaction Feed Object

```
{
  "id": "...",
  "alias": "...",
  "filter": "...",
  "after": "..."
}
```

### Create Transaction Feed

#### Endpoint

```
POST /create-transaction-feed
```

#### Request

```
{
    "alias": "...", // optional
    "filter": "..."
}
```

#### Response

A Transaction Feed object.

### Get Transaction Feed

#### Endpoint

```
POST /get-transaction-feed
```

#### Request

```
{
  // Provide either id or alias
  "id": ...
  "alias": ...
}
```

#### Response

A Transaction Feed object.

### List Transaction Feeds

#### Endpoint

```
POST /list-transaction-feeds
```

#### Request

```
{
  "after": <string> // optional
}
```

#### Response

```
{
  "items": [
    <Transaction Feed object>,
    ...
  ],
  "next": {
    "after": <string>
  }
  "last_page": <boolean>
}
```

### Update Transaction Feed

Updates the Transaction Feed with a new `after`. This is used to acknowledge that the last set of transactions received from `/list-transactions` was processed successfully.

Transaction Feeds can only be updated forwards (i.e., a feed cannot be updated with a value that is previous to its current value).

If present, the `previous_after` field will be used to prevent a race condition where two clients are updating the same feed at the same time. If the current feed does not match `previous_after`, it cannot be updated.

#### Endpoint

```
POST /update-transaction-feed
```

#### Request

```
{
  // Provide either id or alias
  "id": "...",
  "alias": "...",

  "previous_after": "...", // optional
  "after": "..."
}
```

#### Response

The updated Transaction Feed object.

### Delete Transaction Feed

#### Endpoint

```
POST /delete-transaction-feed
```

#### Request

```
{
    // Provide either id or alias
    "id": "..."
    "alias": "..."
}
```

#### Response

```
{
  "message": "ok"
}
```

## Access Tokens

### Create Access Token

#### Endpoint

```
POST /create-access-token
```

#### Request

```
{
  "id": <string>,
  "type": <"client"|"network">
}
```

#### Response

```
{
  "id": <string>,
  "token": "<id>:<secret>",
  "type": <"client"|"network">,
  "created_at": <string, RFC3339 timestamp>
}
```

### List Access Token

#### Endpoint

```
POST /list-access-tokens
```

#### Request

```
{
  "type": <"client"|"network">, // optional, default is blank (no filtering)
  "after": <string>, // optional
  "page_size": <integer> // optional, defaults to 100
}
```

#### Response

```
{
  "items": [
    {
      "id": <string>,
      "type": <"client"|"network">,
      "created_at": <string, RFC3339 timestamp>
    },
    ...
  ],
  "next": {
    "type": <"client"|"network">,
    "after": <string>,
    "page_size": <integer>
  },
  "last_page": <boolean>
}
```

### Delete Access Token

#### Endpoint

```
POST /delete-access-token
```

#### Request

```
{
  "id": <string>
}
```

#### Response

```
{
  "message": "ok"
}
```

## Core

### Configure

Configures the core. Can only be called once between [resets](#reset).

#### Endpoint

```
POST /configure
```

#### Request

```
{
  "is_generator": <boolean>,

  // Supply these if is_generator is false.
  "generator_url": ...,
  "generator_access_token": <string>,
  "blockchain_id": <string>
}
```

#### Response

```
{"message": "ok"}
```

Returns 400 error if the generator URL, generator access token, and/or blockchain ID are bad.

### Info

Returns useful information about this core, including the relative distance between the local block height and the generator's block height.

#### Endpoint

```
POST /info
```

#### Request

(empty)

#### Response

```
{
  "is_configured": <boolean>,
  "configured_at": <string, RFC3339 timestamp>,
  "is_signer": <boolean>,
  "is_generator": <boolean>,
  "generator_url": <string>,
  "generator_access_token": <string>, // secret portion should be obfuscated
  "blockchain_id": <string>,
  "block_height": <integer>,
  "generator_block_height": <integer>,
  "is_production": <boolean>,
  "network_rpc_version": <integer>,
  "core_id": <string>,
  "build_commit": <string>,
  "build_date": <string>
}
```

### Reset

Resets all data in the core, including blockchain data, accounts, assets, and HSM keys.

#### Endpoint

```
POST /reset
```

#### Request

(empty)

#### Response

```
{"message": "ok"}
```
