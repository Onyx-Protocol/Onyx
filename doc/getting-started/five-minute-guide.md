# 5-Minute Guide

## Introduction

This guide will walk you through the basic functions of Chain Core:

* Initialize the SDK
* Create Keys (in the Chain Core MockHSM)
* Initialize the HSM Signer
* Create an Account
* Create an Asset
* Issue Asset units into an Account
* Spend Asset units from one Account to another
* Retire Asset units from an Account

## Initialize the SDK
Create an instance of the SDK, providing the URL of the Chain Core API.

```
Context context = new Context(new URL("http://localhost:8080"));
```

## Create Keys
Create a new key in the MockHSM.

```
MockHsm.Key mainkey = MockHsm.Key.create(context);
```

## Initialize the HSM Signer
To be able to sign transactions, load the key into the HSM Signer, which will communicate with the MockHSM.

```
HsmSigner.addKey(mainkey);
```

## Create an Asset
Create a new asset, providing an alias, key, and quorum. The quorum is the threshold of keys that must sign a transaction issuing units of the asset.

```
new Asset.Builder()
    .setAlias("gold")
    .addRootXpub(mainkey.xpub)
    .setQuorum(1)
    .create(context);
```

## Create an Account
Create an account, providing an alias, key, and quorum. The quorum is the threshold of keys that must sign a transaction to spend asset units controlled by the account.

```
new Account.Builder()
    .setAlias("alice")
    .addRootXpub(mainkey.xpub)
    .setQuorum(1)
    .create(context);
```

Create a second account to interact with the first account.

```
new Account.Builder()
    .setAlias("bob")
    .addRootXpub(mainKey.xpub)
    .setQuorum(1)
    .create(context);
```

## Issue Asset Units
Build, sign, and submit a transaction that issues new units of the `gold` asset into the `alice` account.

```
Transaction.Template issuance = new Transaction.Builder()
    .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(100)
    ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(100)
    ).build(context);
Transaction.submit(context, HsmSigner.sign(issuance));
```

## Spend Asset Units
Build, sign, and submit a transaction that spends units of the `gold` asset from the `alice` account to the `bob` account.

```
Transaction.Template spending = new Transaction.Builder()
    .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
    ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
    ).build(context);
Transaction.submit(context, HsmSigner.sign(spending));
```

## Retire Asset Units
Build, sign, and submit a transaction that retires units of the `gold` asset from the `bob` account.

```
Transaction.Template retirement = new Transaction.Builder()
    .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(5)
    ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("gold")
        .setAmount(5)
    ).build(context);
Transaction.submit(context, HsmSigner.sign(retirement));
```


## Putting it all together
```
import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

import java.net.URL;
import java.util.Arrays;

public class Main {
    public static void main(String[] args) throws Exception {
        Context context = new Context(new URL("http://localhost:8080"));
        MockHsm.Key mainKey = MockHsm.Key.create(context);
        HsmSigner.addKey(mainKey);

        new Account.Builder()
            .setAlias("alice")
            .addRootXpub(mainKey.xpub)
            .setQuorum(1)
            .create(context);

        new Account.Builder()
            .setAlias("bob")
            .addRootXpub(mainKey.xpub)
            .setQuorum(1)
            .create(context);

        new Asset.Builder()
            .setAlias("gold")
            .addRootXpub(mainKey.xpub)
            .setQuorum(1)
            .create(context);

        Transaction.Template issuance = new Transaction.Builder()
            .addAction(new Transaction.Action.Issue()
                .setAssetAlias("gold")
                .setAmount(100)
            ).addAction(new Transaction.Action.ControlWithAccount()
                .setAccountAlias("alice")
                .setAssetAlias("gold")
                .setAmount(100)
            ).build(context);
        Transaction.submit(context, HsmSigner.sign(issuance));

        Transaction.Template spending = new Transaction.Builder()
            .addAction(new Transaction.Action.SpendFromAccount()
                .setAccountAlias("alice")
                .setAssetAlias("gold")
                .setAmount(10)
            ).addAction(new Transaction.Action.ControlWithAccount()
                .setAccountAlias("bob")
                .setAssetAlias("gold")
                .setAmount(10)
            ).build(context);
        Transaction.submit(context, HsmSigner.sign(spending));

        Transaction.Template retirement = new Transaction.Builder()
            .addAction(new Transaction.Action.SpendFromAccount()
                .setAccountAlias("bob")
                .setAssetAlias("gold")
                .setAmount(5)
            ).addAction(new Transaction.Action.Retire()
                .setAssetAlias("gold")
                .setAmount(5)
            ).build(context);
        Transaction.submit(context, HsmSigner.sign(retirement));
    }
}
```

[Download Code](https://s3.amazonaws.com/chain-core/20160902/five-minute-guide.zip)
