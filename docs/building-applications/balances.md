# Balances

## Introduction

The Balance.QueryBuilder class is unique in that it does not return an object in the Chain Core, but rather an aggregate sum over the `amount` fields in a defined list of unspent output objects. By default, this returns a list of balances, summed by `asset_id` and `asset_alias`.

## Sum By

The `setSumBy` method enables more complex summations of balances. For example, if you have a network of counterparty-issued IOUs, you may wish to calculate the account balance of all IOUs from different counterparties that represent the same underlying currency.

## Examples

### List the asset IOU balances in Bank1's account

#### Query

$code ../examples/java/Balances.java account-balance

#### Response

```
[
  {
    "sum_by": {
      "asset_id": "123",
      "asset_alias": "bank2_usd_iou"
    },
    "amount": 100
  },
  {
    "sum_by": {
      "asset_id": "123",
      "asset_alias": "bank3_usd_iou"
    },
    "amount": 200
  },
  {
    "sum_by": {
      "asset_id": "123",
      "asset_alias": "bank4_eur_iou"
    },
    "amount": 400
  },
  {
    "sum_by": {
      "asset_id": "123",
      "asset_alias": "bank5_eur_iou"
    },
    "amount": 500
  }
]
```

### Get the circulation of the Bank 1 USD IOU on the blockchain

#### Query

$code ../examples/java/Balances.java usd-iou-circulation

#### Response

```
[
  {
    "sum_by": {
      "asset_id": "123",
      "asset_alias": "bank1_usd_iou"
    },
    "amount": 500000
  }
]
```


###List the asset IOU balances in Bank1's account, summed by currency:

#### Query

$code ../examples/java/Balances.java account-balance-sum-by-currency

#### Response

```
[
  {
    "sum_by": {
      "asset_definition.currency": "USD"    // bank2_usd_iou + bank3_usd_iou, which both have a value of `USD` for `definition.tags.currency`
    },
    "amount": 300
  }
  {
    "sum_by": {
      "asset_definition.currency": "EUR"    // bank4_eur_iou + bank5_eur_iou, which both have a value of `EUR` for `definition.tags.currency`
    },
    "amount": 900
  }
]
```

[Download Code](../examples/java/Balances.java)
