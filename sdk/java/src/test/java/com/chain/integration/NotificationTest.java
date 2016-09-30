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
    HsmSigner.addKey(key);
    long amount = 1000;
    String alice = "TransactionTest.testTransactionNotification.alice";
    String asset = "TransactionTest.testTransactionNotification.test";
    String consumer = "TransactionTest.testTransactionNotification.consumer";
    String filter = "outputs(account_alias="+alice+")";

    new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1).create(context);

    new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1).create(context);

    Transaction.Consumer cnsmr = Transaction.Consumer.create(context, consumer, filter);
    Transaction.QueryBuilder queryer =
        new Transaction.QueryBuilder().setAfter(cnsmr.after).setAscending().setTimeout(1000);

    ExecutorService executor = Executors.newFixedThreadPool(1);
    Callable<Transaction.Items> task =
        () -> {
          return queryer.execute(context);
        };
    Future<Transaction.Items> future = executor.submit(task);

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

    Transaction.Items txns = future.get();
    assertEquals(txns.list.size(), 1);
    assertEquals(txns.list.get(0).inputs.get(0).action, "issue");
    assertEquals(txns.list.get(0).inputs.get(0).amount, amount);
    assertEquals(txns.list.get(0).outputs.get(0).amount, amount);
  }
}
