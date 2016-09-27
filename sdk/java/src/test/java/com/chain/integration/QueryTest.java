package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.http.Context;
import org.junit.Test;

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
}
