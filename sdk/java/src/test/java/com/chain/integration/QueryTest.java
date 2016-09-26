package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.http.Context;
import org.junit.Test;

import static org.junit.Assert.assertEquals;

public class QueryTest {
  static Context context;
  static MockHsm.Key key;

  @Test
  public void run() throws Exception {
    testAccountQuery();
    testAssetQuery();
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
}
