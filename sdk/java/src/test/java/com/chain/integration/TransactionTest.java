package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.Balance;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.http.BatchResponse;
import com.chain.http.Client;
import com.chain.signing.HsmSigner;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static junit.framework.TestCase.assertNotNull;
import static org.junit.Assert.assertEquals;

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
    new Account.Builder().setAlias(bob).addRootXpub(key.xpub).setQuorum(1).create(client);
    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(client);

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
    Transaction.submit(client, HsmSigner.sign(issuance));

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
    Transaction.submit(client, HsmSigner.sign(spending));

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
  }

  public void testMultiSigTransaction() throws Exception {
    client = TestUtils.generateClient();
    key = MockHsm.Key.create(client);
    key2 = MockHsm.Key.create(client);
    key3 = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    HsmSigner.addKey(key2, MockHsm.getSignerClient(client));
    HsmSigner.addKey(key3, MockHsm.getSignerClient(client));
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

    Transaction.Template aliceIssue =
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

    Transaction.Template bobIssue =
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
    BatchResponse<Transaction.Template> signResponses =
        HsmSigner.signBatch(Arrays.asList(aliceIssue, new Transaction.Template()));
    List<Transaction.Template> templates = signResponses.successes();
    templates.add(bobIssue);
    BatchResponse<Transaction.SubmitResponse> submitResponses =
        Transaction.submitBatch(client, templates);
    assertNotNull(signResponses.errors().get(0));
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
            .execute(client);
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
