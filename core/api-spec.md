# Draft: Chain Core API Spec
This document serves as the canonical source of the HTTP interface to Chain Core.
As the API crystallizes, we will add more thorough descriptions of behaviour and data requirements.

## Table of Contents

* [MockHSM](#mockhsm)
  * [Key Object](#key-object)
  * [Create Key](#create-key)
  * [List Keys](#list-keys)
  * [Sign Transaction Template](#sign-transaction-template)
* [Assets](#assets)
  * [Asset Object](#asset-object)
  * [Create Asset](#create-asset)
  * [Set Asset Tags](#set-asset-tags)
  * [List Assets](#list-assets)
* [Accounts](#accounts)
  * [Account Object](#account-object)
  * [Create Account](#create-account)
  * [Set Account Tags](#set-account-tags)
  * [List Accounts](#list-accounts)
* [Control Programs](#control-programs)
  * [Create Control Program](#create-control-program)
* [Transactions](#transactions)
  * [Transaction Template Object](#transaction-template-object)
  * [Transaction Object](#transaction-object)
  * [Unspent Output Object](#unspent-output-object)
  * [Build Transaction Template](#build-transaction-template)
  * [Submit Transaction Template](#submit-transaction-template)
  * [List Transactions](#list-transactions)
  * [List Balances](#list-balances)
  * [List Unspent Outputs](#list-unspent-outputs)
* [Core](#core)
  * [Configure](#configure)
  * [Info](#info)
  * [Reset](#reset)


## MockHSM

### Key Object
    
```
{
  "alias": "...",
  "xpub": "xpub1.."
}
```

### Create Key
    
Endpoint
```
POST /mockhsm/create-key
```

Request
```
{
  "alias": "..." // optional
}
```

Response: [Key Object](#key-object)

### List Keys
    
Endpoint
```    
POST /mockhsm/list-keys
```

Request

```
{
  "after": "..." // optional
}
```

Response

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

### Sign Transaction Template
    
Endpoint
```    
POST mockhsm/sign-transaction-template
```

Request: An array of transaction template objects.

Response: An array of signed transaction template objects.

    

## Assets

### Asset Object
```
  {
    "id": "...",
    "alias": "...",
    "issuance_program: "...",
    "xpubs": ["xpub"],
    "quorum": 1,
    "definition": {},
    "tags": {}
  }
```

### Create Asset
    
Endpoint 
```    
POST /create-asset
```
    
Request
```
[
  {
    "alias": "...",
    "xpubs": ["..."],
    "quorum": 1,
    "definition: {},
    "tags": {}
  }
]
```

Response: An array of [asset objects](#asset-object).

### Set Asset Tags
Replaces any existing tags.

Endpoint 
```    
POST /set-asset-tags
```

Request
```
{
  "asset_id": "...",      // accepts `asset_id` or `asset_alias`
  "tags": {}
}
```

Response: an [asset object](#asset-object).

### List Assets

Endpoint
```
POST /list-assets
```
    
Request

```
{
  "filter": "...", 
  "filter_params": [], // optional
  "after": "..." // optional
}
```

Response

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

Endpoint
```
POST /archive-asset
```
    
Request

```
{
  asset_id: "...",        // accepts `asset_id` or `asset_alias`
}
```

Response

The response body is empty.

## Accounts

### Account Object
```    
{
  "id": "...",
  "alias": "...",
  "xpubs": ["xpub"],
  "quorum": 1,
  "tags": {}
}
```

### Create Account

Endpoint
```    
POST /create-account
```
    
Request
```
[
  {
    "alias": "...",
    "xpubs": ["xpub"],
    "quorum": 1,
    "tags": {}
  }
]
```    

Response: an array of [account objects](#account-object).

### Set Account Tags
Replaces any existing tags.

Endpoint
```    
POST /set-account-tags
```
    
Request
```    
{
  "account_id": "..."       // accepts `account_id` or `account_alias`
  "tags": {}
}
```

Response: an [account object](#account-object).

### List Accounts

Endpoint
```    
POST /list-accounts
```

Request

```
{
  "filter": "...", // optional
  "filter_params": [], // optional
  "after": "..." // optional
}
```

Response

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

Endpoint
```
POST /archive-account
```
    
Request

```
{
  account_id: "...",    // accepts  `account_id` or `account_alias`
}
```

Response

The response body is empty.

## Control Programs

### Create Control Program

Endpoint
```    
POST /create-control-program
```

Request
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
  account_id: "..."         // accepts `account_id` or `account_alias`
}
```

Response

```
[
  {
    "control_program": "..."
  }
]
```

## Transactions

### Transaction Template Object
```
{
  "unsigned_hex": "...",
  "inputs": [
    {
      "asset_id": "2ed22e7846968aaee500b5ea4b4dfc8bdbe798f32e0737516ab44be4417ff111",
      "amount": 4,
      "position": 0,
      "signature_components": [
        {
          "type": "data",
          "data": "abcd..."
        },
        {
          "type": "signature",
          "quorum": 1,
          "signature_data": "e603d3b8a10fb1714b986393c686fc3ab5f361ec29f94cfd8c7ef3e95e5e44d8",
          "signatures": [
            {
              "xpub": "...",
              "derivation_path": [0,0,2,0,9],
              "signature": ""
            }
          ]
        }
      ]
    }
  ]
}
```

### Transaction Object
Annotated by the Core services where possible (account_ids, account_tags, asset_tags)

```        
    {
        "id": "C5D3F8...",
        "timestamp": "2015-12-30T00:02:23Z",
        "block_id": "A83585...",
        "block_height": 100,
        "position": ...,                                        // position in block
        "reference_data": {"deal_id": ..."},
        "inputs": [
            {
                "action": "issue",
                "asset_id": "125B4E...",
                "asset_alias": "...",
                "asset_tags": {},
                "issuance_program": ...,
                "reference_data": {"details:": "..."},
                "asset_definition": "..."
            },
            {
                "action": spend,
                "spent_output": {
                    "transaction_id": "94C5D3...",
                    "position": 1,
                },
                "account_id": "",
                "account_alias": "...",
                "account_tags": {},
                "asset_id": "125B4E...",
                "asset_alias": "...",
                "asset_tags": {},
                "amount": 5000,
                "reference_data": {"user": "alice"}
            }
        ],
        "outputs": [
            {
                "action": "control",
                "position": "...",
                "account_id": "",
                "account_alias": "...",
                "account_tags": {},
                "asset_id": "125B4E...",
                "asset_alias": "...",
                "asset_tags": {},
                "amount": 5000,
                "control_program": "205CDF...",
                "reference_data": {"user": "bob"}
            }
        ]
    }
```

Note: the "retire" SDK method is simply a control program containing the `RETURN` operation.
To keep the interface narrow, the SDK can generate such a control program.

### Unspent Output Object

```
{
  "transaction_id": "...",
  "position": "...",
  "asset_id": "...",
  "asset_alias": "...",
  "asset_tags": {},
  "amount": 5000,
  "account_id": "...",
  "account_alias": "...",
  "account_tags": {},
  "control_program": "...",
  "reference_data": {}
}
```

### Build Transaction Template

Endpoint
```    
POST /build-transaction-template
```

Request
```
    [
        {  
            "transaction":"...",
            "reference_data":"...",
            "actions":[  
                {
                	"type":"spend_account_unspent_output_selector",
                	"params":{
                		"asset_id":"...",                                 // accepts `asset_id` or `asset_alias`
                		"amount":123,
                		"account_id":"..."
                	}
                },
                {
                	"type":"spend_account_unspent_output",
                	"params":{
                		"transaction_id":"...",
                		"position":0
                	},
                	"reference_data":"..."
                },
                {
                	"type":"issue",
                	"params":{
                		"asset_id":"...",                                 // accepts `asset_id` or `asset_alias`
                		"amount":500
                	},
                	"reference_data":"..."
                },
                {
                	"type":"control_account",
                	"params":{
                		"asset_id":"...",                                 // accepts `asset_id` or `asset_alias`
                		"amount":500,
                		"account_id":"..."                                // accepts `account_id` or `account_alias`
                	},
                	"reference_data":"..."
                },
                {
                	"type":"control_program",
                	"params":{
                		"asset_id":"...",                                 // accepts `asset_id` or `asset_alias`
                		"amount":500,
                		"control_program":"..."
                	},
                	"reference_data":"..."
                }
            ]
        }
    ]
```

Response: An array of [transaction template objects](#transaction-template-object).

### Submit Transaction Template

Endpoint
```
POST /submit-transaction-template
```

Request: an array of [transaction template objects](#transaction-template-object).

Response
```
[
  {
    "id": "..."
  }
]
```

### List Transactions

Endpoint
```
POST /list-transactions
```

Request

```
{
  "filter": "...", // optional
  "filter_params": [], // optional
  "order": <"asc"|"desc">, // optional, defaults to "desc" (newest to oldest)
  "after": "..." // optional
}
```

Response

```
{
  "items": [
    <transaction object>,
    ...
  ],
  "next": {
    "filter": "...",
    "filter_params": [],
    "order": <"asc"|"desc">
    "after": "..."
  },
  "last_page": true|false
}
```

### List Balances

Endpoint
```
POST /list-balances
```

Request

``` 
{
  "filter": "...", // optional
  "filter_params": ["param"], // optional
  "sum_by": ["selector1", ...] // optional
}
```

Response 

Grouped
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
  "last_page": true, // currently only returns one page
  "next": {...}
}
```
    
Ungrouped 
```    
{
  "items": [
    {
      "amount": 10
    }
  ],
  "last_page": true,
  "next": {
    "filter": "...",
    "filter_params": [],
    "sum_by": [...],
    "after": "..."
  },
}
```

### List Unspent Outputs

Endpoint
```
POST /list-unspent-outputs
```

Request

```
{
  "filter": "...", // optional
  "filter_params": [], // optional
  "after": "..." // optional
}
```

Response

```
{
  "items": [
    <unspent output object>,
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

## Core

### Configure

Configures the core. Can only be called once between [resets](#reset).

Endpoint

```
POST /configure
```

Request

```
{
  "is_generator": <true | false>,

  // Supply these if is_generator is false.
  "generator_url": ...,
  "initial_block_hash": ...,
}
```

Response

```
{"message": "ok"}
```

Returns 400 error if the generator URL and/or initial block hash is bad.

### Info

Returns useful information about this core, including the relative distance between the local block height and the generator's block height.

Endpoint

```
POST /info
```

Request

(empty)

Response

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

Endpoint

```
POST /reset
```

Request

(empty)

Reponse

```
{"message": "ok"}
```
