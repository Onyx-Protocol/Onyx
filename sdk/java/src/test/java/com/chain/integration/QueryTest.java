package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;

import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

public class QueryTest {
  static Context context;
  static MockHsm.Key key;

  @Test
  public void run() throws Exception {
    testAccountQuery();
    testAccountPagination();
    testAssetQuery();
    testAssetPagination();
    testTransactionQuery();
  }

  public void testAccountQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String alice = "QueryTest.testAccountQuery.alice";
    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(context);
    Account.Items items =
        new Account.QueryBuilder().setFilter("alias=$1").addFilterParameter(alice).execute(context);
    assertEquals(items.list.size(), 1);
    assertEquals(items.next().alias, alice);
  }

  public void testAccountPagination() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String tag = "QueryTest.testAccountPagination.tag";
    for (int i = 0; i < 101; i++) {
      new Account.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("tag", tag).create(context);
    }

    Account.Items items =
        new Account.QueryBuilder()
            .setFilter("tags.tag=$1")
            .addFilterParameter(tag)
            .execute(context);
    assertEquals(items.list.size(), 100);
    int counter = 0;
    while (items.hasNext()) {
      assertNotNull(items.next().id);
      counter++;
    }
    assertEquals(counter, 101);
  }

  public void testAssetQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String asset = "QueryTest.testAssetQuery.alice";
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(context);
    Asset.Items items =
        new Asset.QueryBuilder().setFilter("alias=$1").addFilterParameter(asset).execute(context);
    assertEquals(items.list.size(), 1);
    assertEquals(items.next().alias, asset);
  }

  public void testAssetPagination() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String tag = "QueryTest.testAssetPagination.tag";
    for (int i = 0; i < 101; i++) {
      new Asset.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("tag", tag).create(context);
    }

    Asset.Items items =
        new Asset.QueryBuilder().setFilter("tags.tag=$1").addFilterParameter(tag).execute(context);
    assertEquals(items.list.size(), 100);
    int counter = 0;
    while (items.hasNext()) {
      assertNotNull(items.next().id);
      counter++;
    }
    assertEquals(counter, 101);
  }

  public void testTransactionQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    HsmSigner.addKey(key);
    String alice = "QueryTest.testTransactionQuery.alice";
    String asset = "QueryTest.testTransactionQuery.asset";
    String test = "QueryTest.testTransactionQuery.test";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(context);
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(context);

    Map<String, Object> refData = new HashMap<>();
    refData.put("asset", asset);
    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .setReferenceData(refData)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .setReferenceData(refData)
                    .addReferenceDataField("test", test))
            .setReferenceData(refData)
            .addReferenceDataField("test", test)
            .build(context);
    Transaction.submit(context, HsmSigner.sign(issuance));
    Transaction.Items txs =
        new Transaction.QueryBuilder()
            .setFilter("inputs(reference_data.test=$1)")
            .addFilterParameter(test)
            .execute(context);
    Transaction tx = txs.next();
    assertEquals(txs.list.size(), 1);
    assertEquals(tx.referenceData.get("asset"), asset);
    assertEquals(tx.referenceData.get("test"), test);

    txs =
        new Transaction.QueryBuilder()
            .setFilter("outputs(reference_data.test=$1)")
            .addFilterParameter(test)
            .execute(context);
    tx = txs.next();
    assertEquals(txs.list.size(), 1);
    assertEquals(tx.referenceData.get("asset"), asset);
    assertEquals(tx.referenceData.get("test"), test);

    txs =
        new Transaction.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .execute(context);
    tx = txs.next();
    assertEquals(txs.list.size(), 1);
    assertEquals(tx.referenceData.get("asset"), asset);
    assertEquals(tx.referenceData.get("test"), test);
  }
}
