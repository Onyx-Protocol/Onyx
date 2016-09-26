package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.exception.APIException;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;
import org.junit.Test;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.List;

public class TransactionTest {
  static Context context;
  static MockHsm.Key key;
  static MockHsm.Key key2;
  static MockHsm.Key key3;

  @Test
  public void run() throws Exception {
    testBasicTransaction();
    testMultiSigTransaction();
  }

  public void testBasicTransaction() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    HsmSigner.addKey(key);
    String alice = "TransactionTest.testBasicTransaction.alice";
    String bob = "TransactionTest.testBasicTransaction.bob";
    String asset = "TransactionTest.testBasicTransaction.asset";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(context);
    new Account.Builder().setAlias(bob).addRootXpub(key.xpub).setQuorum(1).create(context);
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(context);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(100))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(issuance));

    Transaction.Template spending =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(10))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(10))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(spending));

    Transaction.Template retirement =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(5))
            .addAction(new Transaction.Action.Retire().setAssetAlias(asset).setAmount(5))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(retirement));
  }

  public void testMultiSigTransaction() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    key2 = MockHsm.Key.create(context);
    key3 = MockHsm.Key.create(context);
    HsmSigner.addKey(key);
    HsmSigner.addKey(key2);
    HsmSigner.addKey(key3);
    String alice = "TransactionTest.testMultiSigTransaction.alice";
    String bob = "TransactionTest.testMultiSigTransaction.bob";
    String asset = "TransactionTest.testMultiSigTransaction.asset";

    new Account.Builder()
        .setAlias(alice)
        .addRootXpub(key.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(2)
        .create(context);
    new Account.Builder()
        .setAlias(bob)
        .addRootXpub(key.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(1)
        .create(context);
    new Asset.Builder()
        .setAlias(asset)
        .addRootXpub(key.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(1)
        .create(context);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(100))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(issuance));

    Transaction.Template spending =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(10))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(10))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(spending));

    Transaction.Template retirement =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(5))
            .addAction(new Transaction.Action.Retire().setAssetAlias(asset).setAmount(5))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(retirement));
  }
}
