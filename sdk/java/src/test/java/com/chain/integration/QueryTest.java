package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.*;
import com.chain.http.Client;
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
  static Client client;
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
    client = TestUtils.generateClient();
    for (int i = 0; i < 10; i++) {
      MockHsm.Key.create(client, String.format("%d", i));
    }
    MockHsm.Key.Items items =
        new MockHsm.Key.QueryBuilder()
            .setAliases(Arrays.asList("1", "2", "3"))
            .addAlias("4")
            .addAlias("5")
            .execute(client);
    assertEquals(5, items.list.size());
  }

  public void testAccountQuery() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    String alice = "QueryTest.testAccountQuery.alice";
    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(client);
    Account.Items items =
        new Account.QueryBuilder().setFilter("alias=$1").addFilterParameter(alice).execute(client);
    assertEquals(1, items.list.size());
    assertEquals(alice, items.next().alias);
  }

  public void testAssetQuery() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    String asset = "QueryTest.testAssetQuery.alice";
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(client);
    Asset.Items items =
        new Asset.QueryBuilder().setFilter("alias=$1").addFilterParameter(asset).execute(client);
    assertEquals(1, items.list.size());
    assertEquals(asset, items.next().alias);
  }

  public void testAssetPagination() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    String tag = "QueryTest.testAssetPagination.tag";
    for (int i = 0; i < PAGE_SIZE + 1; i++) {
      new Asset.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("tag", tag).create(client);
    }

    Asset.Items items =
        new Asset.QueryBuilder().setFilter("tags.tag=$1").addFilterParameter(tag).execute(client);
    assertEquals(items.list.size(), PAGE_SIZE);
    int counter = 0;
    while (items.hasNext()) {
      assertNotNull(items.next().id);
      counter++;
    }
    assertEquals(counter, PAGE_SIZE + 1);
  }

  public void testTransactionQuery() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String alice = "QueryTest.testTransactionQuery.alice";
    String asset = "QueryTest.testTransactionQuery.asset";
    String test = "QueryTest.testTransactionQuery.test";
    long amount = 100;

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(client);
    Receiver receiver = new Account.ReceiverBuilder().setAccountAlias(alice).create(client);

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
                new Transaction.Action.ControlWithReceiver()
                    .setReceiver(receiver)
                    .setAssetAlias(asset)
                    .setAmount(amount)
                    .setReferenceData(refData)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.SetTransactionReferenceData()
                    .setReferenceData(refData)
                    .addReferenceDataField("test", test))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(issuance));

    Transaction.Items txs =
        new Transaction.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .setStartTime(System.currentTimeMillis())
            .execute(client);
    assertEquals(0, txs.list.size());

    new Transaction.QueryBuilder()
        .setFilter("reference_data.test=$1")
        .addFilterParameter(test)
        .setEndTime(System.currentTimeMillis() - 100000000000L)
        .execute(client);
    assertEquals(0, txs.list.size());

    txs =
        new Transaction.QueryBuilder()
            .setFilter("inputs(reference_data.test=$1)")
            .addFilterParameter(test)
            .execute(client);
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
            .execute(client);
    tx = txs.next();
    assertEquals(1, txs.list.size());
    assertEquals(asset, tx.referenceData.get("asset"));
    assertEquals(test, tx.referenceData.get("test"));

    txs =
        new Transaction.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .execute(client);
    tx = txs.next();
    assertEquals(1, txs.list.size());
    assertEquals(asset, tx.referenceData.get("asset"));
    assertEquals(test, tx.referenceData.get("test"));
  }

  public void testBalanceQuery() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String asset = "QueryTest.testBalanceQuery.asset";
    String alice = "QueryTest.testBalanceQuery.alice";
    String test = "QueryTest.testBalanceQuery.test";
    long amount = 100;

    new Asset.Builder()
        .addRootXpub(key.xpub)
        .setAlias(asset)
        .addTag("name", asset)
        .setQuorum(1)
        .create(client);

    for (int i = 0; i < 10; i++) {
      Account account =
          new Account.Builder()
              .setAlias(alice + i)
              .addRootXpub(key.xpub)
              .setQuorum(1)
              .create(client);
      Transaction.Template issuance =
          new Transaction.Builder()
              .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(amount))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(account.alias)
                      .setAssetAlias(asset)
                      .setAmount(amount)
                      .addReferenceDataField("test", test))
              .build(client);
      Transaction.submit(client, HsmSigner.sign(issuance));
    }

    Balance.Items items =
        new Balance.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .setTimestamp(System.currentTimeMillis() - 100000000000L)
            .execute(client);
    assertEquals(0, items.list.size());

    items =
        new Balance.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .setTimestamp(System.currentTimeMillis())
            .execute(client);
    Balance bal = items.next();
    assertNotNull(bal.sumBy);
    assertNotNull(bal.sumBy.get("asset_alias"));
    assertNotNull(bal.sumBy.get("asset_id"));
    assertEquals(1, items.list.size());
    assertEquals(1000, bal.amount);
  }

  public void testUnspentOutputQuery() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String asset = "QueryTest.testUnspentOutputQuery.asset";
    String alice = "QueryTest.testUnspentOutputQuery.alice";
    String test = "QueryTest.testUnspentOutputQuery.test";
    long amount = 100;

    new Asset.Builder()
        .addRootXpub(key.xpub)
        .setAlias(asset)
        .addTag("name", asset)
        .setQuorum(1)
        .addDefinitionField("name", asset)
        .create(client);

    for (int i = 0; i < 10; i++) {
      Account account =
          new Account.Builder()
              .setAlias(alice + i)
              .addRootXpub(key.xpub)
              .setQuorum(1)
              .addTag("test", test)
              .create(client);
      Transaction.Template issuance =
          new Transaction.Builder()
              .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(amount))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(account.alias)
                      .setAssetAlias(asset)
                      .setAmount(amount)
                      .addReferenceDataField("test", test))
              .build(client);
      Transaction.submit(client, HsmSigner.sign(issuance));
    }
    UnspentOutput.Items items =
        new UnspentOutput.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .setFilterParameters(Arrays.asList((Object) test))
            .setTimestamp(System.currentTimeMillis() - 100000000000L)
            .execute(client);
    assertEquals(0, items.list.size());

    items =
        new UnspentOutput.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .setFilterParameters(Arrays.asList((Object) test))
            .execute(client);
    UnspentOutput unspent = items.next();
    assertNotNull(unspent.type);
    assertNotNull(unspent.purpose);
    assertNotNull(unspent.transactionId);
    assertNotNull(unspent.position);
    assertNotNull(unspent.assetId);
    assertNotNull(unspent.assetAlias);
    assertNotNull(unspent.accountId);
    assertNotNull(unspent.accountAlias);
    assertNotNull(unspent.controlProgram);
    assertNotNull(unspent.assetTags);
    assertNotNull(unspent.assetDefinition);
    assertNotNull(unspent.accountTags);
    assertNotNull(unspent.referenceData);
    assertEquals(100, unspent.amount);
    assertEquals("yes", unspent.isLocal);
    assertEquals("yes", unspent.assetIsLocal);
    assertEquals(asset, unspent.assetDefinition.get("name"));
    assertEquals(10, items.list.size());
  }

  // Because BaseQueryBuilder#getPage is used in the execute
  // method and for pagination, testing pagination for one
  // api object is sufficient for exercising the code path.
  public void testPagination() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    String tag = "QueryTest.testPagination.tag";
    for (int i = 0; i < PAGE_SIZE + 1; i++) {
      new Account.Builder()
          .addRootXpub(key.xpub)
          .setAlias(String.format("%d", i))
          .setQuorum(1)
          .addTag("tag", tag)
          .create(client);
    }

    Account.Items items =
        new Account.QueryBuilder().setFilter("tags.tag=$1").addFilterParameter(tag).execute(client);
    assertEquals(PAGE_SIZE, items.list.size());
    int counter = 0;
    while (items.hasNext()) {
      assertNotNull(items.next().id);
      counter++;
    }
    assertEquals(PAGE_SIZE + 1, counter);
  }
}
