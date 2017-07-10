<!---
An overview of the Asset, Account, Transaction, and Unspent Output API objects.
-->

# API Objects

* [Asset](#asset)
* [Account](#account)
* [Transaction](#transaction)
* [Unspent Output](#unspent-output)

## Asset
### Field Descriptions
The following fields are present in the asset object. Fields with **global** visibility exist as part of the immutable record on the blockchain. Fields with **local** visibility are private annotations that are only visible to the Core.

| Field            | Type        | Visibility          | Description                                                                                  |
|------------------|-------------|---------------------|----------------------------------------------------------------------------------------------|
| id               | string      | global              | Globally unique identifier of the asset.                                                     |
| alias            | string      | local               | User-supplied, locally unique identifier of the asset.                                       |
| issuance_program | string      | global              | The program defining the keys and quorum of signatures required to issue units of the asset. |
| quorum           | integer     | global              | The number of keys from which signatures are required to issue units of the asset.           |
| definition       | JSON&nbsp;object | global              | Arbitrary, user-supplied, key-value data about the asset.                                    |
| tags             | JSON&nbsp;object | local               | Arbitrary, user-supplied, key-value data about the asset.                                    |
| is_local         | string      | local               | Denotes if the asset was created in the Core.                                                |
| keys             | array       | (see&nbsp;[Keys](#keys)) | A list of keys used to generate the `issuance_program`.                                      |

#### Keys
| Field                 | Type   | Visibility | Description                                                                                |
|-----------------------|--------|------------|--------------------------------------------------------------------------------------------|
| asset_pubkey          | string | global     | The public key derived from the `root_xpub` that is present in the `issuance_program`.     |
| root_xpub             | string | local      | The root extended public key provided at time of asset creation.                           |
| asset_derivation_path | array  | local      | The hierarchical deterministic derivation path of the `asset_pubkey` from the `root_xpub`. |


### Example
```
{
  "id": "...",
  "alias": "...",
  "issuance_program: "...",
  "keys": [
    {
      "asset_pubkey": "...",
      "root_xpub": "...",
      "asset_derivation_path": "..."
    }
  ],
  "quorum": 1,
  "definition": {},
  "tags": {},
  "is_local": <"yes"|"no">
}
```

## Account
### Field Descriptions
The following fields are present in the account object. This object is local to Chain Core and not visible on the blockchain. Only the derived, one-time-use public keys and quorum in each account control program are visible on the blockchain (see [transaction output](#output)).

| Field  | Type        | Description                                                                                  |
|--------|-------------|----------------------------------------------------------------------------------------------|
| id     | string      | Locally unique identifier of the account.                                                    |
| alias  | string      | User-supplied, locally unique identifier of the account.                                     |
| quorum | integer     | The number of keys from which signatures are required to spent asset units from the account. |
| tags   | JSON&nbsp;object | Arbitrary, user-supplied, key-value data about the account.                                  |
| keys   | array       | A list of keys used to generate control programs in the account.                             |

#### Keys

| Field                 | Type   | Description                                                                                                               |
|-----------------------|--------|---------------------------------------------------------------------------------------------------------------------------|
| root_xpub             | string | The root extended public key provided at time of account creation.                                                        |
| asset_derivation_path | array  | The hierarchical deterministic derivation path of the `account_xpub` from the `root_xpub`.                                |
| account_xpub          | string | The extended public key derived from the `root_xpub` from which pubkeys for each control program will be further derived. |

### Example

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

## Transaction
### Field Descriptions
The following fields are present in the transaction object. Fields with **global** visibility exist as part of the immutable record on the blockchain. Fields with **local** visibility are private annotations that are only visible to the Core.

| Field          | Type        | Visibility | Description                                                                                                                                                                                        |
|----------------|-------------|------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| id             | string      | global     | Globally unique identifier of the transaction.                                                                                                                                                     |
| timestamp      | string      | global     | Time (in RFC3339 format) that the transaction was committed to the blockchain.                                                                                                                     |
| is_local       | string      | local      | Denotes if the Core was involved in the transaction, either by: a) issuing units an asset created in the Core, b) spending from an account in the Core, or c) receiving to an account in the Core. |
| block_id       | string      | global     | The globally unique identifier of the block in which the transaction was committed to the blockchain.                                                                                              |
| block_height   | integer     | global     | The sequential number of the block in which the transaction was committed to the blockchain.                                                                                                       |
| position       | integer     | global     | The sequential number of the transaction in the block in which the transaction was committed to the blockchain.                                                                                    |
| inputs         | array       | global     | A list of source(s) of asset units in the transaction.                                                                                                                                             |
| outputs        | array       | global     | A list of destination(s) of asset units in the transaction.                                                                                                                                        |
| reference_data | JSON&nbsp;object | global     | Arbitrary, user-supplied, key-value data about the transaction.                                                                                                                                    |

#### Input

| Field          | Type        | Visibility | Description                                                                                                                                  |
|----------------|-------------|------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| type           | string      | global     | Type of input - either `issuance` or `spending`.                                                                                             |
| is_local       | string      | local      | Denotes that the input involves the Core, either by: a) issuing units an asset created in the Core, b) spending from an account in the Core. |
| asset_id       | string      | global     | The cryptographic, globally unique identifier of the asset being issued or spent.                                                            |
| asset_alias    | string      | local      | User-supplied, locally unique identifier of the asset being issued or spent.                                                                 |
| asset_tags     | JSON&nbsp;object | local      | Arbitrary, user-supplied, key-value data about the asset being issued or spent.                                                              |
| asset_is_local | string      | local      | Denotes if the asset being issued or spent was created in the Core.                                                                          |
| amount         | integer     | global     | Amount of units of the asset being issued or spent.                                                                                          |
| reference_data | JSON&nbsp;object | global     | Arbitrary, user-supplied, key-value data about the input.                                                                                    |

#### Input (if `type` is `spending`)

| Field         | Type        | Visibility | Description                                                                          |
|---------------|-------------|------------|--------------------------------------------------------------------------------------|
| account_id    | string      | local      | Locally unique identifier of the account spending the asset units.                   |
| account_alias | string      | local      | User-supplied, locally unique identifier of the account spending the asset units.    |
| account_tags  | string      | local      | Arbitrary, user-supplied, key-value data about the account spending the asset units. |
| spent_output_id | string    | global     | The ID of the previous transaction output being spent in the input.                  |

#### Output

| Field           | Type        | Visibility | Description                                                                                                                                  |
|-----------------|-------------|------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| id              | string      | global     | The unique ID of the output.                                                                                                                 |
| type            | string      | global     | Type of output - either `control` or `retirement`.                                                                                            |
| is_local        | string      | local      | Denotes that the input involves the Core, either by: a) issuing units an asset created in the Core, b) spending from an account in the Core. |
| purpose         | string      | local      | Purpose of the output - either a) `receive` if used to receive asset units from another account or external party, or b) `change` if used to create change back to the account, when spending only a portion of the amount of an unspent output in a "spending" input.|
| position        | integer     | global     | The sequential number of the output in the transaction.                                                                                      |
| asset_id        | string      | global     | The cryptographic, globally unique identifier of the asset being controlled or retired.                                                      |
| asset_alias     | string      | local      | User-supplied, locally unique identifier of the asset being controlled or retired.                                                           |
| asset_tags      | JSON&nbsp;object | local      | Arbitrary, user-supplied, key-value data about the asset being controlled or retired.                                                        |
| asset_is_local  | string      | local      | Denotes if the asset being controlled or retired was created in the Core.                                                                    |
| amount          | integer     | global     | Amount of units of the asset being controlled or retired.                                                                                    |
| reference_data  | JSON&nbsp;object | global     | Arbitrary, user-supplied, key-value data about the output.                                                                                   |
| control_program | string      | global     | The program that controls the asset units in the output.                                                                                     |

#### Output (if `type` is `control`)

| Field         | Type   | Visibility | Description                                                                             |
|---------------|--------|------------|-----------------------------------------------------------------------------------------|
| account_id    | string | local      | Locally unique identifier of the account controlling the asset units.                   |
| account_alias | string | local      | User-supplied, locally unique identifier of the account controlling the asset units.    |
| account_tags  | string | local      | Arbitrary, user-supplied, key-value data about the account controlling the asset units. |

### Example

```
{
  "id": "C5D3F8...",
  "timestamp": "2015-12-30T00:02:23Z",
  "block_id": "3d6732d...",
  "block_height": 100,
  "position": ..., // position in block
  "reference_data": {"deal_id": "..."},
  "is_local": <"yes"|"no">, // local if any input or output is local
  "inputs": [
    {
      "action": "issue",
      "asset_id": "125b4e...",
      "asset_alias": "...",
      "asset_tags": {},
      "asset_is_local": <"yes"|"no">
      "amount": 5000,
      "issuance_program": ...,
      "reference_data": {"details": "..."},
      "asset_definition": "...",
      "is_local": <"yes"|"no"> // local if action is issue and asset is local
    },
    {
      "action": "spend",
      "asset_id": "125b4e...",
      "asset_alias": "...",
      "asset_tags": {},
      "asset_is_local": <"yes"|"no">,
      "amount": 5000,
      "spent_output_id": "997de5...",
      "account_id": "",
      "account_alias": "...",
      "account_tags": {},
      "reference_data": {"user": "alice"},
      "is_local": <"yes"|"no"> // local if account id is not null
    }
  ],
  "outputs": [
    {
      "action": "control",
      "purpose": <"change"|"receive">, // provided if the control program was generated locally
      "id": "311df2...",
      "position": "...",
      "asset_id": "125b4e...",
      "asset_alias": "...",
      "asset_tags": {},
      "asset_is_local": <"yes"|"no">,
      "amount": 6000,
      "account_id": "",
      "account_alias": "...",
      "account_tags": {},
      "control_program": "205CDF...",
      "reference_data": {"user": "bob"},
      "is_local": <"yes"|"no"> // local if action is control and account id is not null
    },
    {
      "action": "retire",
      "id": "2eb5cf...",
      "position": "...",
      "asset_id": "125b4e...",
      "asset_alias": "...",
      "asset_tags": {},
      "asset_is_local": <"yes"|"no">,
      "amount": 4000,
      "account_id": "",
      "account_alias": "...",
      "account_tags": {},
      "control_program": "6a",
      "reference_data": {"user": "bob"},
      "is_local": <"yes"|"no"> // local if action is control and account id is not null
    }
  ]
}
```

## Unspent Output
### Field Descriptions
The unspent output object is a subset of the [transaction object](#transaction). It includes all the fields present in the [output](#output) of a transaction, with the addition of the `transaction_id` of the transaction in which it is contained.

### Example

```
{
  "id": "2eb5cf...",
  "action": "control",
  "purpose": <"change"|"receive">
  "transaction_id": "...",
  "position": "...",
  "asset_id": "...",
  "asset_alias": "...",
  "asset_definition": {},
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
