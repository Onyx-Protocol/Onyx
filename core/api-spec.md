# Draft: Chain Core API Spec
This document serves as the canonical source of the HTTP interface to Chain Core.
As the API crystallizes, we will add more thorough descriptions of behaviour and data requirements.

## Table of Contents

* [MockHSM](#mockhsm)
  * [Key Object](#key-object)
  * [Create Key](#create-key)
  * [Get Key](#get-key)
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
  * [Create Transaction](#create-transaction)
  * [List Transactions](#list-transactions)
  * [List Balances](#list-balances)
  * [List Unspent Outputs](#list-unspent-outputs)
* [Indexes](#indexes)
  * [Index Object](#index-object)
  * [Create Index](#create-index)
  * [Get Index](#get-index)
  * [List Indexes](#list-indexes)


## MockHSM

### Key Object
    
```
{
  "id": "...",
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
  "id": "..."     // user supplied. must be unique. if not provided, core will create.
}
```

Response: [Key Object](#key-object)

### Get Key
    
Endpoint
```
POST /mockhsm/get-key
```    

Request
```
{
  "id": "..."
}
```

Response: [Key Object](#key-object)

### List Keys
    
Endpoint
```    
POST /mockhsm/list-keys
```

Response: An array of [key objects](#key-object).

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
    "asset_id": "...",
    "issuance_program: "...",
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
  "asset_id": "...",
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
  index_id: "...",
  params: ["param"],
}
```

Response: an array of [asset objects](#asset-object).

## Accounts

### Account Object
```    
{
  "id": "...",
  "xpubs": ["xpub"],
  "quorum": 1,
  "tags": {}
}
```

### Create Account
Creates one or more new accounts.

Endpoint
```    
POST /create-account
```
    
Request
```
[
  {
    "id": "...",          // user supplied. must be unique. if not provided, core will create.
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
  "account_id": "..."
  "tags": {}
}
```

Response: an [account object](#account-object).

### List Accounts
Optionally filtered by a query.

Endpoint
```    
POST /list-accounts
```

Request
```    
{
  "query": "accounts_tags.entity = $1 AND accounts_tags.bank = $2"
  "params": ["acme", "bank1"]
}
```

Response: an array of [account objects](#account-object).

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

If the `type` is `account` then the following params are require:
```
{
  account_id: "..."
}
```

## Transactions

### Transaction Template Object
NOTE: Revisit this! What are the implications on `redemption_predicates` when there are many types of control programs? Should there be a `control_program_type`?

```
{
  "inputs": [
    {
      "asset_id": "2ed22e7846968aaee500b5ea4b4dfc8bdbe798f32e0737516ab44be4417ff111",
      "amount": 4,
      "redemption_predicates": "255121031351da7e482a17720fa36154d739ec204faa9dc488d286d9e0126049e5ddea4951ae",      // revisit this! should there by a `control_program_type`?
      "signature_data": "e603d3b8a10fb1714b986393c686fc3ab5f361ec29f94cfd8c7ef3e95e5e44d8",
      "signatures": [
        {
          "xpub_hash": "...",
          "derivation_path": [0,0,2,0,9],
          "signature": ""
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
                "asset_tags": {},
                "issuance_program": ...,
                "reference_data": {"details:": "..."},
                "asset_definition": "..."                       // if provided at issuance
            },
            {
                "action": spend,
                "spent_output": {
                    "transaction_id": "94C5D3...",
                    "position": 1,
                },
                "account_ids": [],
                "account_tags": {},
                "asset_id": "125B4E...",
                "asset_tags": {},
                "amount": 5000,
                "reference_data": {"user": "alice"}
            }
        ],
        "outputs": [
            {
                "action": "control",
                "position": "...",
                "account_ids": [],
                "account_tags": {},
                "asset_id": "125B4E...",
                "asset_tags": {},
                "amount": 5000,
                "control_program": "205CDF...",
                "reference_data": {"user": "bob"}
            },
            {
                "action": "destroy",
                "position": "...",
                "asset_id": "125B4E...",
                "asset_tags": {},
                "amount": 1000,
                "control_program": "OP_RETURN",
                "reference_data": {"wire_transfer_id": "..."}
            }
        ]
    }
```

### Unspent Output Object

```
{
  "transaction_id": "...",
  "position": "...",
  "asset_id": "...",
  "asset_tags": {},
  "amount": 5000,
  "account_id": "...",
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
                		"asset_id":"...",
                		"amount":123,
                		"account_id":"..."
                	}
                },
                {
                	"type":"spend_account_unspent_output",
                	"params":{
                		"account_id":"...",
                		"transaction_id":"...",
                		"position":0
                	},
                	"reference_data":"..."
                },
                {
                	"type":"issue",
                	"params":{
                		"asset_id":"...",
                		"amount":500
                	},
                	"reference_data":"..."
                },
                {
                	"type":"control_account",
                	"params":{
                		"asset_id":"...",
                		"amount":500,
                		"account_id":"..."
                	},
                	"reference_data":"..."
                },
                {
                	"type":"control_program",
                	"params":{
                		"asset_id":"...",
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

### Create Transaction

Endpoint
```
POST /create-transaction
```

Request: an array of [transaction template objects](#transaction-template-object).

Response: an array of [transaction objects](#transaction-object).

### List Transactions

Endpoint
```
POST /list-transactions
```

Request

Accepts either an `index_id` or a `query`.
```
{
  "index_id": "...",
  "query": "...,"
  "params": ["param"],
}
```

Response: an array of [transaction objects](#transaction-object).

### List Balances

Endpoint
```
POST /list-balances
```

Request

Accepts either an `index_id` or a `query`.

``` 
{
  "index_id": "...",
  "query": "...,"
  "params": ["fishco", null],      // wildcard is group_by
}
```

Response 

Grouped
```    
[
  {
    "asset_id": "a1",       
    "amount": 10
  },
  {
    "asset_id": "a2",
    "amount": 20
  }
]
```
    
Ungrouped 
```    
[
  {
    "amount": 10
  }
]
```

### List Unspent Outputs

Endpoint
```
POST /list-unspent-outputs
```

Request

Accepts either an `index_id` or a `query`.
```
{
  "index_id": "...",
  "query": "...,"
  "params": ["param"],
}
```

Response: an array of [output objects](#output-object).

## Indexes

### Index Object
```    
{
  "id": "...",            
  "type": "...",              // `transaction`, `balance`, or `asset`
  "unspents": "true",         // only for `type: "balance"` - indexes unspent outputs in addition to balances
  "query": "..."
}
```

### Create Index

Endpoint
```
POST /create-index
```

Request
```
{
  "id": "...",            // user supplied. must be unique. if not provided, core will create.
  "type": "...",          // `transaction`, `balance`, or `asset`
  "unspents": "true",     // only for `type: "balance"` - indexes unspent outputs in addition to balances
  "query": "..."
}
```
Response: an [index object](#index-object).

### Get Index

Endpoint
```
POST /get-index
```

Request
```    
{
  "id": "..."
}
```

Response: an [index object](#index-object).

### List Indexes

Endpoint
```
POST /list-indexes
```

Response: an array of [index objects](#index-object).
