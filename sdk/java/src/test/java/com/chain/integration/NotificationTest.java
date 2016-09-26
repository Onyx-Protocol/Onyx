package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.Cursor;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.exception.APIException;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;
import org.junit.Test;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.concurrent.Callable;
import java.util.concurrent.Executors;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.Assert.*;

public class NotificationTest {
  final int AMOUNT = 100;
  final String ALICE = "notif-alice";
  final String ASSET = "notif-asset";
  final String CURSOR = "notif-cursor";
  final String FILTER = "outputs(account_alias='"+ALICE+"')";

  @Test
  public void test() throws Exception {
    Context context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    MockHsm.Key mainKey = MockHsm.Key.create(context);
    HsmSigner.addKey(mainKey);

    new Account.Builder()
        .setAlias(ALICE)
        .addRootXpub(mainKey.xpub)
        .setQuorum(1)
        .create(context);

    new Asset.Builder()
        .setAlias(ASSET)
        .addRootXpub(mainKey.xpub)
        .setQuorum(1)
        .create(context);

    Cursor cur = Cursor.create(context, CURSOR, FILTER);
    Transaction.QueryBuilder queryer = new Transaction.QueryBuilder()
        .setAfter(cur.after)
        .setAscending()
        .setTimeout(1000);

    ExecutorService executor = Executors.newFixedThreadPool(1);
    Callable<Transaction.Items> task = () -> {
      return queryer.execute(context);
    };
    Future<Transaction.Items> future = executor.submit(task);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(ASSET).setAmount(AMOUNT))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(ALICE)
                    .setAssetAlias(ASSET)
                    .setAmount(AMOUNT))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(issuance));

    Transaction.Items txns = future.get();
    assertEquals(txns.list.size(), 1);
    assertEquals(txns.list.get(0).inputs.get(0).action, "issue");
    assertEquals(txns.list.get(0).inputs.get(0).amount, AMOUNT);
    assertEquals(txns.list.get(0).outputs.get(0).amount, AMOUNT);
  }
}