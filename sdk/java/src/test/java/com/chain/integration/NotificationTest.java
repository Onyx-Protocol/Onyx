package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;

import com.chain.http.Context;
import com.chain.signing.HsmSigner;
import org.junit.Test;

import java.util.concurrent.Callable;
import java.util.concurrent.Executors;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;

import static org.junit.Assert.*;

public class NotificationTest {
  static Context context;
  static MockHsm.Key key;

  @Test
  public void run() throws Exception {
    testTransactionNotification();
  }

  public void testTransactionNotification() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    HsmSigner.addKey(key, MockHsm.getSignerContext(context));
    long amount = 1000;
    String alice = "TransactionTest.testTransactionNotification.alice";
    String asset = "TransactionTest.testTransactionNotification.test";
    String feed = "TransactionTest.testTransactionNotification.feed";
    String filter = "outputs(account_alias='" + alice + "')";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(context);

    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(context);

    Transaction.Feed txfeed = Transaction.Feed.create(context, feed, filter);
    ExecutorService executor = Executors.newFixedThreadPool(1);
    Callable<Transaction> task = () -> txfeed.next(context);
    Future<Transaction> future = executor.submit(task);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(asset).setAmount(amount))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(alice)
                    .setAssetAlias(asset)
                    .setAmount(amount))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(issuance));

    Transaction tx = future.get();
    assertEquals(tx.inputs.get(0).type, "issue");
    assertEquals(tx.inputs.get(0).amount, amount);
    assertEquals(tx.outputs.get(0).amount, amount);
  }
}
