# Query Filters

## Overview

Filters enable complex queries to Chain Core to retrieve lists of objects. A filter is comprised of one or more **terms** (joined with AND / OR). Each term contains a **property**, **operator**, and **value**. Each term targets a specific field in the key-value (JSON) object (see [API Objects](../reference/api-objects)). Terms can be grouped together in a **scope** to target a specific array of sub-objects within an object.

For example, if you wish list transactions where a specific account spends a specific asset, you would create a filter with two terms, scoped to the inputs:

```
inputs(account_alias='alice' AND asset_alias='gold')
```


## Properties

Each method accepts filters specific to the key-value (JSON) object returned. Any key can be used as a filter property. To use a key that is nested within other keys, provide the path to the object, including the parent objects. For example:

```
asset_definition.issuer.name='Acme'
```

## Operators

Filter terms currently support only the `=` operator.

## Scope

The transaction object is the only object that contains an array of other objects - an `inputs` array and an `outputs` array. The `inputs()` and `outputs()` scopes allow targeting a specific object within those arrays.

For example, the following will return transactions where Alice sent gold to Bob.

```
inputs(account_alias='alice' AND asset_alias='gold') AND outputs(account_alias='bob' AND asset_alias='gold')
```

## Queries

The following QueryBuilder classes support filters:

* Transaction.QueryBuilder
* Account.QueryBuilder
* Asset.QueryBuilder
* UnspentOutput.QueryBuilder
* Balance.QueryBuilder

### Note about Balance.QueryBuilder

The Balance.QueryBuilder class is unique in that it does not return an object in the Chain Core, but rather a sum over the `amount` fields a in defined list of unspents outputs. By default, this returns a single `amount`, but can be grouped using a `sumBy` parameter in addition to the filter.

For example, to list all balances for `alice`, summed by `asset_alias`, you would set the following filter:

```
account_alias='alice'
```

and set `sumBy` to:

```
asset_alias
```

which will return the following:

```
[
    {
        "sum_by": {"asset_alias": "gold"},
        "amount": 10
    },
    {
        "sum_by": {"asset_alias": "silver"},
        "amount": 20
    }
]
```
