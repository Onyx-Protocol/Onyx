package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.Balance;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.api.UnspentOutput;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;

import org.junit.Test;

import java.util.Arrays;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertTrue;

public class QueryTest {
  static Context context;
  static MockHsm.Key key;
  final static int PAGE_SIZE = 100;

  @Test
  public void run() throws Exception {
    testKeyQuery();
    testAccountQuery();
    testAssetQuery();
    testTransactionQuery();
    testBalanceQuery();
    testUnspentOutputQuery();
    testPagination();
  }

  public void testKeyQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    for (int i = 0; i < 10; i++) {
      MockHsm.Key.create(context, String.format("%d", i));
    }
    MockHsm.Key.Items items =
        new MockHsm.Key.QueryBuilder()
            .setAliases(Arrays.asList("1", "2", "3"))
            .addAlias("4")
            .addAlias("5")
            .execute(context);
    assertEquals(5, items.list.size());
  }

  public void testAccountQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String alice = "QueryTest.testAccountQuery.alice";
    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(context);
    Account.Items items =
        new Account.QueryBuilder().setFilter("alias=$1").addFilterParameter(alice).execute(context);
    assertEquals(1, items.list.size());
    assertEquals(alice, items.next().alias);
  }

  public void testAssetQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String asset = "QueryTest.testAssetQuery.alice";
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(context);
    Asset.Items items =
        new Asset.QueryBuilder().setFilter("alias=$1").addFilterParameter(asset).execute(context);
    assertEquals(1, items.list.size());
    assertEquals(asset, items.next().alias);
  }

  public void testAssetPagination() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String tag = "QueryTest.testAssetPagination.tag";
    for (int i = 0; i < PAGE_SIZE + 1; i++) {
      new Asset.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("tag", tag).create(context);
    }

    Asset.Items items =
        new Asset.QueryBuilder().setFilter("tags.tag=$1").addFilterParameter(tag).execute(context);
    assertEquals(items.list.size(), PAGE_SIZE);
    int counter = 0;
    while (items.hasNext()) {
      assertNotNull(items.next().id);
      counter++;
    }
    assertEquals(counter, PAGE_SIZE + 1);
  }

  public void testTransactionQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    HsmSigner.addKey(key, MockHsm.getSignerContext(context));
    String alice = "QueryTest.testTransactionQuery.alice";
    String asset = "QueryTest.testTransactionQuery.asset";
    String test = "QueryTest.testTransactionQuery.test";
    long amount = 100;

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(context);
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(context);

    Map<String, Object> refData = new HashMap<>();
    refData.put("asset", asset);
    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetAlias(asset)
                    .setAmount(amount)
                    .setReferenceData(refData)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(amount)
                    .setReferenceData(refData)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.SetTransactionReferenceData()
                    .setReferenceData(refData)
                    .addReferenceDataField("test", test))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(issuance));

    Transaction.Items txs =
        new Transaction.QueryBuilder()
            .setFilter("inputs(reference_data.test=$1)")
            .addFilterParameter(test)
            .execute(context);
    Transaction tx = txs.next();
    assertNotNull(tx.id);
    assertNotNull(tx.blockId);
    assertNotNull(tx.blockHeight);
    assertNotNull(tx.inputs);
    assertNotNull(tx.outputs);
    assertNotNull(tx.position);
    assertNotNull(tx.timestamp);
    assertTrue(tx.timestamp.before(new Date()));
    assertEquals("yes", tx.isLocal);
    assertEquals(1, tx.inputs.size());
    assertEquals(1, tx.outputs.size());
    assertEquals(1, txs.list.size());
    assertEquals(asset, tx.referenceData.get("asset"));
    assertEquals(test, tx.referenceData.get("test"));

    txs =
        new Transaction.QueryBuilder()
            .setFilter("outputs(reference_data.test=$1)")
            .addFilterParameter(test)
            .execute(context);
    tx = txs.next();
    assertEquals(1, txs.list.size());
    assertEquals(asset, tx.referenceData.get("asset"));
    assertEquals(test, tx.referenceData.get("test"));

    txs =
        new Transaction.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .execute(context);
    tx = txs.next();
    assertEquals(1, txs.list.size());
    assertEquals(asset, tx.referenceData.get("asset"));
    assertEquals(test, tx.referenceData.get("test"));
  }

  public void testBalanceQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    HsmSigner.addKey(key, MockHsm.getSignerContext(context));
    String asset = "QueryTest.testBalanceQuery.asset";
    String alice = "QueryTest.testBalanceQuery.alice";
    String test = "QueryTest.testBalanceQuery.test";
    long amount = 100;

    new Asset.Builder()
        .addRootXpub(key.xpub)
        .setAlias(asset)
        .addTag("name", asset)
        .setQuorum(1)
        .create(context);

    for (int i = 0; i < 10; i++) {
      Account account =
          new Account.Builder()
              .setAlias(alice + i)
              .addRootXpub(key.xpub)
              .setQuorum(1)
              .create(context);
      Transaction.Template issuance =
          new Transaction.Builder()
              .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(amount))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(account.alias)
                      .setAssetAlias(asset)
                      .setAmount(amount)
                      .addReferenceDataField("test", test))
              .build(context);
      Transaction.submit(context, HsmSigner.sign(issuance));
    }

    Balance.Items items =
        new Balance.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .execute(context);
    Balance bal = items.next();
    assertNotNull(bal.sumBy);
    assertNotNull(bal.sumBy.get("asset_alias"));
    assertNotNull(bal.sumBy.get("asset_id"));
    assertEquals(1, items.list.size());
    assertEquals(1000, bal.amount);
  }

  public void testUnspentOutputQuery() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    HsmSigner.addKey(key, MockHsm.getSignerContext(context));
    String asset = "QueryTest.testUnspentOutputQuery.asset";
    String alice = "QueryTest.testUnspentOutputQuery.alice";
    String test = "QueryTest.testUnspentOutputQuery.test";
    long amount = 100;

    new Asset.Builder()
        .addRootXpub(key.xpub)
        .setAlias(asset)
        .addTag("name", asset)
        .setQuorum(1)
        .create(context);

    for (int i = 0; i < 10; i++) {
      Account account =
          new Account.Builder()
              .setAlias(alice + i)
              .addRootXpub(key.xpub)
              .setQuorum(1)
              .create(context);
      Transaction.Template issuance =
          new Transaction.Builder()
              .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(amount))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(account.alias)
                      .setAssetAlias(asset)
                      .setAmount(amount)
                      .addReferenceDataField("test", test))
              .build(context);
      Transaction.submit(context, HsmSigner.sign(issuance));
    }

    UnspentOutput.Items items =
        new UnspentOutput.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .execute(context);
    UnspentOutput unspent = items.next();
    assertNotNull(unspent.purpose);
    assertNotNull(unspent.transactionId);
    assertNotNull(unspent.position);
    assertNotNull(unspent.amount);
    assertEquals("control", unspent.type);
    assertEquals(10, items.list.size());
  }

  // Because BaseQueryBuilder#getPage is used in the execute
  // method and for pagination, testing pagination for one
  // api object is sufficient for exercising the code path.
  public void testPagination() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String tag = "QueryTest.testPagination.tag";
    for (int i = 0; i < PAGE_SIZE + 1; i++) {
      new Account.Builder()
          .addRootXpub(key.xpub)
          .setAlias(String.format("%d", i))
          .setQuorum(1)
          .addTag("tag", tag)
          .create(context);
    }

    Account.Items items =
        new Account.QueryBuilder()
            .setFilter("tags.tag=$1")
            .addFilterParameter(tag)
            .execute(context);
    assertEquals(PAGE_SIZE, items.list.size());
    int counter = 0;
    while (items.hasNext()) {
      assertNotNull(items.next().id);
      counter++;
    }
    assertEquals(PAGE_SIZE + 1, counter);
  }
}
