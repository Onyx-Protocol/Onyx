package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.*;
import com.chain.exception.*;
import com.chain.http.*;

import org.junit.Test;

import java.util.*;

import static org.junit.Assert.*;

public class UpdateTagsTest {
  @Test
  public void accountTags() throws Exception {
    Client client = TestUtils.generateClient();
    MockHsm.Key key = MockHsm.Key.create(client);

    Account account1 =
        new Account.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("x", "zero").create(client);
    Account account2 =
        new Account.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("y", "zero").create(client);

    Map<String, Object> update1, update2;

    // Account tag update

    update1 = new HashMap<>();
    update1.put("x", "one");

    new Account.TagUpdateBuilder().forId(account1.id).setTags(update1).update(client);

    Account.Items accounts =
        new Account.QueryBuilder()
            .setFilter("id=$1")
            .addFilterParameter(account1.id)
            .execute(client);

    while (accounts.hasNext()) {
      assertEquals(accounts.next().tags.get("x"), "one");
    }

    // Account tag update that raises an error

    try {
      update1 = new HashMap<>();
      update1.put("x", "two");

      new Account.TagUpdateBuilder()
          // ID intentionally omitted
          .setTags(update1)
          .update(client);
    } catch (APIException e) {
      assertTrue(e.toString().contains("CH051"));
    }

    // Account tag batch update

    update1 = new HashMap<>();
    update1.put("x", "three");
    update2 = new HashMap<>();
    update2.put("y", "three");

    BatchResponse<SuccessMessage> batch =
        Account.updateTagsBatch(
            client,
            Arrays.asList(
                new Account.TagUpdateBuilder().forId(account1.id).setTags(update1),
                new Account.TagUpdateBuilder().forId(account2.id).setTags(update2)));

    assertEquals(batch.errorsByIndex().size(), 0);

    accounts =
        new Account.QueryBuilder()
            .setFilter("id=$1 OR id=$2")
            .addFilterParameter(account1.id)
            .addFilterParameter(account2.id)
            .execute(client);

    List<Account> accountList = new ArrayList<>();
    while (accounts.hasNext()) {
      accountList.add(accounts.next());
    }

    // Results from a query are returned in reverse-chronological order by creation time.
    assertEquals(accountList.get(0).tags.get("y"), "three");
    assertEquals(accountList.get(1).tags.get("x"), "three");

    // Account tag batch update with error

    update1 = new HashMap<>();
    update1.put("x", "four");
    update2 = new HashMap<>();
    update2.put("y", "four");

    batch =
        Account.updateTagsBatch(
            client,
            Arrays.asList(
                new Account.TagUpdateBuilder().forId(account1.id).setTags(update1),
                new Account.TagUpdateBuilder()
                    // ID intentionally omitted
                    .setTags(update2)));

    assertTrue(batch.isSuccess(0));
    assertTrue(batch.isError(1));
  }

  @Test
  public void assetTags() throws Exception {
    Client client = TestUtils.generateClient();
    MockHsm.Key key = MockHsm.Key.create(client);

    Asset asset1 =
        new Asset.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("x", "zero").create(client);
    Asset asset2 =
        new Asset.Builder().addRootXpub(key.xpub).setQuorum(1).addTag("y", "zero").create(client);

    Map<String, Object> update1, update2;

    // Asset tag update

    update1 = new HashMap<>();
    update1.put("x", "one");

    new Asset.TagUpdateBuilder().forId(asset1.id).setTags(update1).update(client);

    Asset.Items assets =
        new Asset.QueryBuilder().setFilter("id=$1").addFilterParameter(asset1.id).execute(client);

    while (assets.hasNext()) {
      assertEquals(assets.next().tags.get("x"), "one");
    }

    // Asset tag update that raises an error

    try {
      update1 = new HashMap<>();
      update1.put("x", "two");

      new Asset.TagUpdateBuilder()
          // ID intentionally omitted
          .setTags(update1)
          .update(client);
    } catch (APIException e) {
      assertTrue(e.toString().contains("CH051"));
    }

    // Asset tag batch update

    update1 = new HashMap<>();
    update1.put("x", "three");
    update2 = new HashMap<>();
    update2.put("y", "three");

    BatchResponse<SuccessMessage> batch =
        Asset.updateTagsBatch(
            client,
            Arrays.asList(
                new Asset.TagUpdateBuilder().forId(asset1.id).setTags(update1),
                new Asset.TagUpdateBuilder().forId(asset2.id).setTags(update2)));

    assertEquals(batch.errorsByIndex().size(), 0);

    assets =
        new Asset.QueryBuilder()
            .setFilter("id=$1 OR id=$2")
            .addFilterParameter(asset1.id)
            .addFilterParameter(asset2.id)
            .execute(client);

    List<Asset> assetList = new ArrayList<>();
    while (assets.hasNext()) {
      assetList.add(assets.next());
    }

    // Results from a query are returned in reverse-chronological order by creation time.
    assertEquals(assetList.get(0).tags.get("y"), "three");
    assertEquals(assetList.get(1).tags.get("x"), "three");

    // Asset tag batch update with error

    update1 = new HashMap<>();
    update1.put("x", "four");
    update2 = new HashMap<>();
    update2.put("y", "four");

    batch =
        Asset.updateTagsBatch(
            client,
            Arrays.asList(
                new Asset.TagUpdateBuilder().forId(asset1.id).setTags(update1),
                new Asset.TagUpdateBuilder()
                    // ID intentionally omitted
                    .setTags(update2)));

    assertTrue(batch.isSuccess(0));
    assertTrue(batch.isError(1));
  }
}
