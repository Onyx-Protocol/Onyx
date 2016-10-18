package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.ControlProgram;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.exception.APIException;
import com.chain.http.BatchResponse;
import com.chain.http.Context;

import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

public class CreateTest {
  static Context context;
  static MockHsm.Key key;

  @Test
  public void run() throws Exception {
    testKeyCreate();
    testAccountCreate();
    testAccountCreateBatch();
    testAssetCreate();
    testAssetCreateBatch();
    testControlProgramCreate();
    testControlProgramCreateBatch();
    testTransactionFeedCreate();
  }

  public void testKeyCreate() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    String alias = "CreateTest.testKeyCreate.alias";
    key = MockHsm.Key.create(context, alias);
    assertNotNull(key.xpub);
    assertEquals(alias, key.alias);

    try {
      MockHsm.Key.create(context, alias);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testAccountCreate() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String alice = "CreateTest.testAccountCreate.alice";
    Account account =
        new Account.Builder()
            .setAlias(alice)
            .addRootXpub(key.xpub)
            .setQuorum(1)
            .addTag("name", alice)
            .create(context);
    assertNotNull(account.id);
    assertNotNull(account.keys);
    assertEquals(1, account.keys.length);
    assertNotNull(account.keys[0].accountXpub);
    assertNotNull(account.keys[0].rootXpub);
    assertNotNull(account.keys[0].derivationPath);
    assertEquals(alice, account.alias);
    assertEquals(1, account.quorum);
    assertEquals(alice, account.tags.get("name"));

    try {
      new Account.Builder()
          .setAlias(alice)
          .addRootXpub(key.xpub)
          .setQuorum(1)
          .addTag("name", alice)
          .create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testAccountCreateBatch() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String alice = "CreateTest.testAccountCreateBatch.alice";
    Account.Builder builder =
        new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1);

    Account.Builder failure =
        new Account.Builder().setAlias(alice).addRootXpub(key.xpub).setQuorum(1);

    BatchResponse<Account> resp = Account.createBatch(context, Arrays.asList(builder, failure));
    assertEquals(1, resp.successes().size());
    assertEquals(1, resp.errors().size());
  }

  public void testAssetCreate() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String asset = "CreateTest.testAssetCreate.asset";
    String test = "CreateTest.testAssetCreate.test";
    Map<String, Object> def = new HashMap<>();
    def.put("name", asset);
    Asset testAsset =
        new Asset.Builder()
            .setAlias(asset)
            .addRootXpub(key.xpub)
            .setQuorum(1)
            .addTag("name", asset)
            .setDefinition(def)
            .addDefinitionField("test", test)
            .create(context);
    assertNotNull(testAsset.id, testAsset.issuanceProgram);
    assertNotNull(testAsset.issuanceProgram);
    assertNotNull(testAsset.keys);
    assertEquals(1, testAsset.keys.length);
    assertNotNull(testAsset.keys[0].assetPubkey);
    assertNotNull(testAsset.keys[0].rootXpub);
    assertNotNull(testAsset.keys[0].derivationPath);
    assertEquals(asset, testAsset.alias);
    assertEquals(1, testAsset.quorum);
    assertEquals(asset, testAsset.tags.get("name"));
    assertEquals(asset, testAsset.definition.get("name"));
    assertEquals(test, testAsset.definition.get("test"));
    assertEquals("yes", testAsset.isLocal);

    try {
      new Asset.Builder()
          .setAlias(asset)
          .addRootXpub(key.xpub)
          .setQuorum(1)
          .addTag("name", asset)
          .setDefinition(def)
          .addDefinitionField("test", test)
          .create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testAssetCreateBatch() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String asset = "CreateTest.testAssetCreateBatch.asset";
    Asset.Builder builder = new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1);

    Asset.Builder failure = new Asset.Builder().setAlias(asset).addRootXpub(key.xpub).setQuorum(1);

    BatchResponse<Asset> resp = Asset.createBatch(context, Arrays.asList(builder, failure));
    assertEquals(1, resp.successes().size());
    assertEquals(1, resp.errors().size());
  }

  public void testControlProgramCreate() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String alice = "CreateTest.testControlProgramCreate.alice";
    Account account =
        new Account.Builder()
            .setAlias(alice)
            .addRootXpub(key.xpub)
            .setQuorum(1)
            .addTag("name", alice)
            .create(context);

    ControlProgram ctrlp =
        new ControlProgram.Builder().controlWithAccountById(account.id).create(context);
    assertNotNull(ctrlp.program);

    ctrlp = new ControlProgram.Builder().controlWithAccountByAlias(account.alias).create(context);
    assertNotNull(ctrlp.program);

    try {
      new ControlProgram.Builder().controlWithAccountById("bad-id").create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testControlProgramCreateBatch() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String alice = "CreateTest.testControlProgramCreateBatch.alice";
    Account account =
        new Account.Builder()
            .setAlias(alice)
            .addRootXpub(key.xpub)
            .setQuorum(1)
            .addTag("name", alice)
            .create(context);

    ControlProgram.Builder builder =
        new ControlProgram.Builder().controlWithAccountById(account.id);

    ControlProgram.Builder failure = new ControlProgram.Builder().controlWithAccountById("bad-id");

    BatchResponse<ControlProgram> resp =
        ControlProgram.createBatch(context, Arrays.asList(builder, failure));
    assertEquals(1, resp.successes().size());
    assertEquals(1, resp.errors().size());
  }

  public void testTransactionFeedCreate() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    String alias = "CreateTest.testFeedCreate.feed";
    String filter = "outputs(account_alias='alice')";
    Transaction.Feed feed = Transaction.Feed.create(context, alias, filter);
    assertNotNull(feed.id);
    assertNotNull(feed.after);
    assertEquals(alias, feed.alias);
    assertEquals(filter, feed.filter);
  }
}
