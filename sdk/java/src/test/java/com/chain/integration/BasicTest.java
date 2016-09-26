package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.*;

public class BasicTest {
  final String ALICE = "basic-alice";
  final String BOB = "basic-bob";
  final String ASSET = "basic-asset";
  final String TEST = "basic";

  @Test
  public void test() throws Exception {
    Context context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    MockHsm.Key mainKey = MockHsm.Key.create(context);
    HsmSigner.addKey(mainKey);

    Account account =
        new Account.Builder()
            .setAlias(ALICE)
            .addRootXpub(mainKey.xpub)
            .setQuorum(1)
            .addTag("name", ALICE)
            .addTag("test", TEST)
            .create(context);
    // TODO(boymanjor): Find better test to assert asset creation
    assertNotNull(account.id);
    assertEquals(account.alias, ALICE);
    assertNotNull(account.keys);
    assertEquals(account.keys.length, 1);
    assertNotNull(account.keys[0].accountXpub);
    assertNotNull(account.keys[0].rootXpub);
    assertNotNull(account.keys[0].derivationPath);
    assertEquals(account.quorum, 1);
    assertEquals(account.tags.get("name"), ALICE);
    assertEquals(account.tags.get("test"), TEST);

    new Account.Builder().setAlias(BOB).addRootXpub(mainKey.xpub).setQuorum(1).create(context);

    Map<String, Object> def = new HashMap<>();
    def.put("name", ASSET);
    def.put("test", TEST);
    Asset asset =
        new Asset.Builder()
            .setAlias(ASSET)
            .addRootXpub(mainKey.xpub)
            .setQuorum(1)
            .addTag("name", ASSET)
            .addTag("test", TEST)
            .setDefinition(def)
            .create(context);
    // TODO: Find better test to assert account creation
    assertNotNull(asset.id);
    assertEquals(asset.alias, ASSET);
    assertNotNull(asset.issuanceProgram);
    assertNotNull(asset.keys);
    assertEquals(asset.keys.length, 1);
    assertNotNull(asset.keys[0].assetPubkey);
    assertNotNull(asset.keys[0].rootXpub);
    assertNotNull(asset.keys[0].derivationPath);
    assertEquals(asset.quorum, 1);
    assertEquals(asset.tags.get("name"), ASSET);
    assertEquals(asset.tags.get("test"), TEST);
    assertEquals(asset.definition.get("name"), ASSET);
    assertEquals(asset.definition.get("test"), TEST);
    assertEquals(asset.origin, "local");

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias(ASSET).setAmount(100))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(ALICE)
                    .setAssetAlias(ASSET)
                    .setAmount(100))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(issuance));

    Transaction.Template spending =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(ALICE)
                    .setAssetAlias(ASSET)
                    .setAmount(10))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias(BOB)
                    .setAssetAlias(ASSET)
                    .setAmount(10))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(spending));

    Transaction.Template retirement =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(BOB)
                    .setAssetAlias(ASSET)
                    .setAmount(5))
            .addAction(new Transaction.Action.Retire().setAssetAlias(ASSET).setAmount(5))
            .build(context);
    Transaction.submit(context, HsmSigner.sign(retirement));
  }
}
