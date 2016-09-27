package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.exception.APIException;
import com.chain.http.Context;

import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

public class CreateTest {
  static Context context;
  static MockHsm.Key key;

  @Test
  public void run() throws Exception {
    testAccountCreateSuccess();
    testAccountCreateFailure();
    testAssetCreateSuccess();
    testAssetCreateFailure();
  }

  public void testAccountCreateSuccess() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String alice = "CreateTest.testAccountCreateSuccess.alice";
    Account account =
        new Account.Builder()
            .setAlias(alice)
            .addRootXpub(key.xpub)
            .setQuorum(1)
            .addTag("name", alice)
            .create(context);
    assertNotNull(account.id);
    assertEquals(account.alias, alice);
    assertNotNull(account.keys);
    assertEquals(account.keys.length, 1);
    assertNotNull(account.keys[0].accountXpub);
    assertNotNull(account.keys[0].rootXpub);
    assertNotNull(account.keys[0].derivationPath);
    assertEquals(account.quorum, 1);
    assertEquals(account.tags.get("name"), alice);
  }

  public static void testAccountCreateFailure() throws Exception {
    try {
      context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
      Account account = new Account.Builder().setQuorum(1).create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testAssetCreateSuccess() throws Exception {
    context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    key = MockHsm.Key.create(context);
    String asset = "CreateTest.testAssetCreateSuccess.asset";
    String test = "CreateTest.testAssetCreateSuccess.test";
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
    assertNotNull(testAsset.id);
    assertEquals(testAsset.alias, asset);
    assertNotNull(testAsset.issuanceProgram);
    assertNotNull(testAsset.keys);
    assertEquals(testAsset.keys.length, 1);
    assertNotNull(testAsset.keys[0].assetPubkey);
    assertNotNull(testAsset.keys[0].rootXpub);
    assertNotNull(testAsset.keys[0].derivationPath);
    assertEquals(testAsset.quorum, 1);
    assertEquals(testAsset.tags.get("name"), asset);
    assertEquals(testAsset.definition.get("name"), asset);
    assertEquals(testAsset.definition.get("test"), test);
    assertEquals(testAsset.isLocal, "yes");
  }

  public static void testAssetCreateFailure() throws Exception {
    try {
      context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
      Asset asset = new Asset.Builder().setQuorum(1).create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }
}
