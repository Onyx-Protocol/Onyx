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
* [Cursors](#cursors)
  * [Cursor Object](#cursor-object)
  * [Create Cursor](#create-cursor)
  * [Get Cursor](#get-cursor)
  * [Update Cursor](#update-cursor)
* [Core](#core)
  * [Configure](#configure)
  * [Info](#info)
  * [Reset](#reset)


## Errors

### Error Object

```
{
  "code": <string>,
  "message": <string>
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
  "transactions": [{...}],
  "xpubs": ["..."],
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
      "asset_pubkey": "...",

      // These properties are only available for assets whose origin is local.
      "root_xpub": "...",
      "asset_derivation_path": "..."
    },
    ...
  ],
  "quorum": 1,
  "definition": {},
  "tags": {},
  "origin": <"local"|"external">
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

### Archive Asset

#### Endpoint

```
POST /archive-asset
```

#### Request

```
{
  asset_id: "...", // accepts `asset_id` or `asset_alias`
}
```

#### Response

The response body is empty.

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

### Archive Account

#### Endpoint

```
POST /archive-account
```

#### Request

```
{
  "account_id": "...", // accepts `account_id` or `account_alias`
}
```

#### Response

The response body is empty.

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
  "inputs": [
    {
      "action": "issue",
      "asset_id": "125B4E...",
      "asset_alias": "...",
      "asset_tags": {},
      "asset_origin": <"local"|"external">
      "amount": 5000,
      "issuance_program": ...,
      "reference_data": {"details": "..."},
      "asset_definition": "..."
    },
    {
      "action": "spend",
      "asset_id": "125B4E...",
      "asset_alias": "...",
      "asset_tags": {},
      "asset_origin": <"local"|"external">,
      "amount": 5000,
      "spent_output": {
        "transaction_id": "94C5D3...",
        "position": 1,
      },
      "account_id": "",
      "account_alias": "...",
      "account_tags": {},
      "reference_data": {"user": "alice"}
    }
  ],
  "outputs": [
    {
      "action": "control",
      "purpose": <"change"|"receive">, // provided if the control program was generated locally
      "position": "...",
      "asset_id": "125B4E...",
      "asset_alias": "...",
      "asset_tags": {},
      "asset_origin": <"local"|"external">,
      "amount": 5000,
      "account_id": "",
      "account_alias": "...",
      "account_tags": {},
      "control_program": "205CDF...",
      "reference_data": {"user": "bob"}
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
  "action": "control",
  "purpose": <"change"|"receive">, // provided if the control program was generated locally
  "transaction_id": "...",
  "position": "...",
  "asset_id": "...",
  "asset_alias": "...",
  "asset_tags": {},
  "asset_origin": <"local"|"external">,
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
    "raw_transaction": <hex string>, // optional. an unsubmitted transaction to which additional actions can be appended.
    "reference_data": "...",
    "ttl": <number of milliseconds>, // optional, defaults to 300000 (5 minutes)
    "actions":[
      {
        "type": "spend_account",
        "asset_id": "...", // accepts `asset_id` or `asset_alias`
        "amount": 123,
        "account_id": "..."
      },
      {
        "type": "spend_account_unspent_output",
        "transaction_id": "...",
        "position": 0,
        "reference_data": "..."
      },
      {
        "type": "issue",
        "asset_id": "...", // accepts `asset_id` or `asset_alias`
        "amount": 500,
        "reference_data": "..."
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
  "order": <"asc"|"desc">, // optional, defaults to "desc" (newest to oldest)
  "after": "...", // optional
  "timeout": <number, in milliseconds> // optional, defaults to 1000 (1 second)
}
```

If order is "asc", the request will stay open until there are transactions to return, or until the timeout occurs, whichever happens first.

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
    "order": <"asc"|"desc">
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

## Cursors

To receive crash-resistant notifications about new transactions, Cursors can be used in conjunction with the `/list-transactions` endpoint.

To process new transactions, a client should:

1. Create a Cursor, adding a `filter` and optionally adding an `alias`; or get a previously created Cursor by its id or alias.
2. Extract the `after` from the response.
3. List transactions with the extracted `after` and `filter`, and set `"order": "asc"` on the request body, to receive transactions in the order that they happened.
4. Extract the new `after` from the `/list-transactions` response body.
5. Update the Cursor with the new `after`.
6. Repeat steps 3 - 5.

### Cursor Object

```
{
  "id": "...",
  "alias": "...",
  "filter": "...",
  "order": "..."
  "after": "...",
}
```

### Create Cursor

#### Endpoint

```
POST /create-cursor
```

#### Request

```
{
    "alias": "...", // optional
    "filter": "..."
}
```

#### Response

A Cursor object.

### Get Cursor

#### Endpoint

```
POST /get-cursor
```

#### Request

```
{
  "id": ..., // provide either cursor id or alias
  "alias": ...
}
```

#### Response

A Cursor object.

### Update Cursor

Updates the Cursor with a new `after`. This is used to acknowledge that the last set of transactions received from `/list-transactions` was processed successfully.

#### Endpoint

```
POST /update-cursor
```

#### Request

```
{
    "id": "...", // provide either cursor id or alias
    "alias": "...",
    "after": "..."
}
```

#### Response

```
{"message": "ok"}
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
  "is_generator": <true | false>,

  // Supply these if is_generator is false.
  "generator_url": ...,
  "initial_block_hash": ...,
}
```

#### Response

```
{"message": "ok"}
```

Returns 400 error if the generator URL and/or initial block hash is bad.

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
  "is_configured": <true | false>,
  "configured_at": ...,
  "is_signer": <true | false>,
  "is_generator": <true | false>,
  "generator_url": ...,
  "initial_block_hash": ...,
  "block_height": ...,
  "generator_block_height": ...,
  "is_production": <true | false>,
  "build_commit": ...,
  "build_date": ...
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
