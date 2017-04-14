package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.*;
import com.chain.http.BatchResponse;
import com.chain.http.Client;
import com.chain.signing.HsmSigner;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static junit.framework.TestCase.assertNotNull;
import static junit.framework.TestCase.assertTrue;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNull;

public class TransactionTest {
  static Client client;
  static MockHsm.Key key;
  static MockHsm.Key key2;
  static MockHsm.Key key3;

  @Test
  public void run() throws Exception {
    testBasicTransaction();
    testMultiSigTransaction();
    testBatchTransaction();
    testAtomicSwap();
    testReceivers();
    testUnspentOutputs();
  }

  public void testBasicTransaction() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String alice = "TransactionTest.testBasicTransaction.alice";
    String bob = "TransactionTest.testBasicTransaction.bob";
    String asset = "TransactionTest.testBasicTransaction.asset";
    String test = "TransactionTest.testBasicTransaction.test";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Account.Builder()
        .setAlias(bob)
        .setRootXpubs(Arrays.asList(key.xpub))
        .setQuorum(1)
        .create(client);
    new Asset.Builder()
        .setAlias(asset)
        .setRootXpubs(Arrays.asList(key.xpub))
        .setQuorum(1)
        .create(client);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.SetTransactionReferenceData()
                    .addReferenceDataField("test", test))
            .build(client);
    Transaction.SubmitResponse resp = Transaction.submit(client, HsmSigner.sign(issuance));
    Transaction.Items txs =
        new Transaction.QueryBuilder()
            .setFilter("id=$1")
            .addFilterParameter(resp.id)
            .execute(client);
    Transaction tx = txs.next();
    assertEquals(1, txs.list.size());
    assertNotNull(tx.inputs);
    assertEquals(1, tx.inputs.size());
    assertNotNull(tx.outputs);
    assertEquals(1, tx.inputs.size());

    Transaction.Input input = tx.inputs.get(0);
    assertEquals("issue", input.type);
    assertNotNull(input.assetId);
    assertNotNull(input.assetAlias);
    assertNotNull(input.assetDefinition);
    assertNotNull(input.assetTags);
    assertEquals("yes", input.assetIsLocal);
    assertNull(input.accountId);
    assertNull(input.accountAlias);
    assertNull(input.accountTags);
    assertEquals("yes", input.isLocal);
    assertNull(input.spentOutputId);
    assertNotNull(input.issuanceProgram);
    assertNotNull(input.referenceData);

    Transaction.Output output = tx.outputs.get(0);
    assertEquals("control", output.type);
    assertEquals("receive", output.purpose);
    assertNotNull(output.position);
    assertNotNull(output.assetId);
    assertNotNull(output.assetAlias);
    assertNotNull(output.assetDefinition);
    assertNotNull(output.assetTags);
    assertEquals("yes", output.assetIsLocal);
    assertNotNull(output.accountId);
    assertNotNull(output.accountAlias);
    assertNotNull(output.accountTags);
    assertNotNull(output.controlProgram);
    assertEquals("yes", output.isLocal);
    assertNotNull(output.referenceData);

    Transaction.Template spending =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(10)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(10)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.SetTransactionReferenceData()
                    .addReferenceDataField("test", test))
            .build(client);
    resp = Transaction.submit(client, HsmSigner.sign(spending));
    txs =
        new Transaction.QueryBuilder()
            .setFilter("id=$1")
            .addFilterParameter(resp.id)
            .execute(client);
    tx = txs.next();
    assertEquals(1, txs.list.size());
    assertNotNull(tx.inputs);
    assertTrue(tx.inputs.size() > 0);

    input = tx.inputs.get(0);
    assertNotNull(input.spentOutputId);

    Transaction.Template retirement =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(5)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.Retire()
                    .setAssetAlias(asset)
                    .setAmount(5)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.SetTransactionReferenceData()
                    .addReferenceDataField("test", test))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(retirement));
    txs =
        new Transaction.QueryBuilder()
            .setFilter("reference_data.test=$1")
            .addFilterParameter(test)
            .execute(client);
    assertEquals(3, txs.list.size());
  }

  public void testMultiSigTransaction() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    key2 = MockHsm.Key.create(client);
    key3 = MockHsm.Key.create(client);
    HsmSigner.addKeys(Arrays.asList(key, key2, key3), MockHsm.getSignerClient(client));
    String alice = "TransactionTest.testMultiSigTransaction.alice";
    String bob = "TransactionTest.testMultiSigTransaction.bob";
    String asset = "TransactionTest.testMultiSigTransaction.asset";

    new Account.Builder()
        .setAlias(alice)
        .addRootXpub(key.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(2)
        .create(client);
    new Account.Builder()
        .setAlias(bob)
        .addRootXpub(key.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(1)
        .create(client);
    new Asset.Builder()
        .setAlias(asset)
        .addRootXpub(key.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(1)
        .create(client);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(100))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(issuance));

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
            .build(client);
    Transaction.submit(client, HsmSigner.sign(spending));

    Transaction.Template retirement =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(5))
            .addAction(new Transaction.Action.Retire().setAssetAlias(asset).setAmount(5))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(retirement));
  }

  public void testBatchTransaction() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String alice = "TransactionTest.testBatchTransaction.alice";
    String bob = "TransactionTest.testBatchTransaction.bob";
    String asset = "TransactionTest.testBatchTransaction.asset";
    String test = "TransactionTest.testBatchTransaction.test";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Account.Builder().setAlias(bob).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(client);

    Transaction.Builder aliceBuilder =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.SetTransactionReferenceData()
                    .addReferenceDataField("test", test));

    Transaction.Builder bobTheBuilder =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.SetTransactionReferenceData()
                    .addReferenceDataField("test", test));
    BatchResponse<Transaction.Template> buildResponses =
        Transaction.buildBatch(client, Arrays.asList(aliceBuilder, bobTheBuilder));
    assertEquals(2, buildResponses.successes().size());

    BatchResponse<Transaction.Template> signResponses =
        HsmSigner.signBatch(
            Arrays.asList(buildResponses.successes().get(0), new Transaction.Template()));
    List<Transaction.Template> templates = signResponses.successes();
    templates.add(buildResponses.successes().get(1));

    BatchResponse<Transaction.SubmitResponse> submitResponses =
        Transaction.submitBatch(client, templates);
    assertNotNull(signResponses.errors().get(0));
    assertEquals(2, submitResponses.size());
    assertTrue(submitResponses.isSuccess(0) || submitResponses.isSuccess(1));
    assertTrue(submitResponses.isError(0) || submitResponses.isError(1));
    assertNotNull(submitResponses.successes().get(0));
    assertNotNull(submitResponses.errors().get(0));
  }

  public void testAtomicSwap() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String alice = "TransactionTest.testAtomicSwap.alice";
    String bob = "TransactionTest.testAtomicSwap.bob";
    String gold = "TransactionTest.testAtomicSwap.gold";
    String silver = "TransactionTest.testAtomicSwap.silver";
    String test = "TransactionTest.testAtomicSwap.test";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Account.Builder().setAlias(bob).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Asset.Builder().setAlias(gold).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Asset.Builder().setAlias(silver).addRootXpub(key.xpub).setQuorum(1).create(client);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetAlias(gold)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetAlias(silver)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(gold)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(silver)
                    .setAmount(100)
                    .addReferenceDataField("test", test))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(issuance));

    Transaction.Template swap =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(gold)
                    .setAmount(45)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(silver)
                    .setAmount(80)
                    .addReferenceDataField("test", test))
            .build(client);
    swap = HsmSigner.sign(swap.allowAdditionalActions());
    swap =
        new Transaction.Builder(swap.rawTransaction)
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(silver)
                    .setAmount(80)
                    .addReferenceDataField("test", test))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(gold)
                    .setAmount(45)
                    .addReferenceDataField("test", test))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(swap));

    Balance.Items balances =
        new Balance.QueryBuilder()
            .setFilter("account_alias=$1")
            .addFilterParameter(alice)
            .setSumBy(Arrays.asList("asset_alias"))
            .execute(client);
    assertEquals(2, balances.list.size());
    // Asserts that we are only summing by 1 parameter
    assertEquals(1, balances.list.get(0).sumBy.size());
    Map<String, Long> aliceBalances = createBalanceMap(balances);
    assertEquals(55, aliceBalances.get(gold).intValue());
    assertEquals(80, aliceBalances.get(silver).intValue());

    balances =
        new Balance.QueryBuilder()
            .setFilter("account_alias=$1")
            .addFilterParameter(bob)
            .execute(client);
    Map<String, Long> bobBalances = createBalanceMap(balances);
    assertEquals(45, bobBalances.get(gold).intValue());
    assertEquals(20, bobBalances.get(silver).intValue());
  }

  public void testReceivers() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String alice = "TransactionTest.testReceivers.alice";
    String bob = "TransactionTest.testReceivers.bob";
    String asset = "TransactionTest.testReceivers.asset";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Account.Builder()
        .setAlias(bob)
        .setRootXpubs(Arrays.asList(key.xpub))
        .setQuorum(1)
        .create(client);
    new Asset.Builder()
        .setAlias(asset)
        .setRootXpubs(Arrays.asList(key.xpub))
        .setQuorum(1)
        .create(client);
    Receiver bobReceiver = new Account.ReceiverBuilder().setAccountAlias(bob).create(client);
    String bobJsonReceiver =
        new Account.ReceiverBuilder().setAccountAlias(bob).create(client).toJson();

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(100))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(issuance));

    Transaction.Template spending =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAssetAlias(asset)
                    .setAccountAlias(alice)
                    .setAmount(10))
            .addAction(
                new Transaction.Action.ControlWithReceiver()
                    .setReceiver(bobReceiver)
                    .setAssetAlias(asset)
                    .setAmount(10))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(spending));

    Transaction.Template spending2 =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAssetAlias(asset)
                    .setAccountAlias(alice)
                    .setAmount(10))
            .addAction(
                new Transaction.Action.ControlWithReceiver()
                    .setReceiver(Receiver.fromJson(bobJsonReceiver))
                    .setAssetAlias(asset)
                    .setAmount(10))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(spending2));

    Balance.Items balances =
        new Balance.QueryBuilder()
            .setFilter("account_alias=$1")
            .addFilterParameter(alice)
            .execute(client);
    assertEquals(1, balances.list.size());
    Map<String, Long> aliceBalances = createBalanceMap(balances);
    assertEquals(80, aliceBalances.get(asset).intValue());
    balances =
        new Balance.QueryBuilder()
            .setFilter("account_alias=$1")
            .addFilterParameter(bob)
            .execute(client);
    Map<String, Long> bobBalances = createBalanceMap(balances);
    assertEquals(20, bobBalances.get(asset).intValue());
  }

  public void testUnspentOutputs() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    String alice = "TransactionTest.testUnspentOutputs.alice";
    String bob = "TransactionTest.testUnspentOutputs.bob";
    String asset = "TransactionTest.testUnspentOutputs.asset";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Account.Builder()
        .setAlias(bob)
        .setRootXpubs(Arrays.asList(key.xpub))
        .setQuorum(1)
        .create(client);
    new Asset.Builder()
        .setAlias(asset)
        .setRootXpubs(Arrays.asList(key.xpub))
        .setQuorum(1)
        .create(client);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(100))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(100))
            .build(client);
    Transaction.SubmitResponse resp = Transaction.submit(client, HsmSigner.sign(issuance));

    Transaction.Items txs =
        new Transaction.QueryBuilder()
            .setFilter("id=$1")
            .addFilterParameter(resp.id)
            .execute(client);
    Transaction tx = txs.next();
    Transaction.Output output = tx.outputs.get(0);

    Transaction.Template spending =
        new Transaction.Builder()
            .addAction(new Transaction.Action.SpendAccountUnspentOutput().setOutputId(output.id))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(bob)
                    .setAssetAlias(asset)
                    .setAmount(10))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(90))
            .build(client);
    resp = Transaction.submit(client, HsmSigner.sign(spending));

    UnspentOutput.Items items =
        new UnspentOutput.QueryBuilder()
            .setFilter("transaction_id=$1")
            .addFilterParameter(resp.id)
            .execute(client);
    UnspentOutput unspent = items.next();

    Transaction.Template retirement =
        new Transaction.Builder()
            .addAction(new Transaction.Action.SpendAccountUnspentOutput().setUnspentOutput(unspent))
            .addAction(
                new Transaction.Action.Retire().setAssetAlias(asset).setAmount(unspent.amount))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(retirement));
  }

  private static Map<String, Long> createBalanceMap(Balance.Items balances) {
    Map<String, Long> balanceMap = new HashMap<>();
    while (balances.hasNext()) {
      Balance balance = balances.next();
      String asset = balance.sumBy.get("asset_alias");
      long x;
      if (balanceMap.containsKey(asset)) {
        x = balanceMap.get(asset);
      } else {
        x = 0;
      }
      balanceMap.put(asset, x + balance.amount);
    }
    return balanceMap;
  }
}
