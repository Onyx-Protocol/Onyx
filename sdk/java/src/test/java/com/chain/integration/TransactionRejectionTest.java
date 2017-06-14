package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.*;
import com.chain.http.Client;
import com.chain.signing.HsmSigner;
import com.chain.exception.APIException;

import java.util.*;

import org.junit.Test;

/**
 * TransactionRejectionTest checks various transaction-rejected error
 * conditions.
 */
public class TransactionRejectionTest {
  static Client client;

  static Transaction issuanceTx;
  static MockHsm.Key keyAlice;
  static MockHsm.Key keyBob;
  static MockHsm.Key keyGold;
  static MockHsm.Key keySilver;
  static MockHsm.Key keyBronze;
  static final Queue<Transaction.Output> outputsAliceGold = new LinkedList<>();
  static final Queue<Transaction.Output> outputsBobSilver = new LinkedList<>();
  static final Queue<Transaction.Output> outputsAliceBronze = new LinkedList<>();
  static final String aliasAlice = "TransactionRejectionTest.alice";
  static final String aliasBob = "TransactionRejectionTest.bob";
  static final String aliasGold = "TransactionRejectionTest.gold";
  static final String aliasSilver = "TransactionRejectionTest.silver";
  static final String aliasBronze = "TransactionRejectionTest.bronze";

  @Test
  public void run() throws Exception {
    setup();
    testSubmitUnbalanced();
    testSubmitUnbalancedTwoAssets();
    testSubmitUnsignedSpend();
    testSubmitUnsignedIssuance();
    testSubmitPartiallySigned();
  }

  public void setup() throws Exception {
    client = TestUtils.generateClient();
    keyAlice = MockHsm.Key.create(client);
    keyBob = MockHsm.Key.create(client);
    keyGold = MockHsm.Key.create(client);
    keySilver = MockHsm.Key.create(client);
    keyBronze = MockHsm.Key.create(client);
    HsmSigner.addKey(keyAlice, MockHsm.getSignerClient(client));
    HsmSigner.addKey(keyBob, MockHsm.getSignerClient(client));
    HsmSigner.addKey(keyGold, MockHsm.getSignerClient(client));
    HsmSigner.addKey(keySilver, MockHsm.getSignerClient(client));
    HsmSigner.addKey(keyBronze, MockHsm.getSignerClient(client));

    new Account.Builder()
        .setAlias(aliasAlice)
        .addRootXpub(keyAlice.xpub)
        .setQuorum(1)
        .create(client);
    new Account.Builder().setAlias(aliasBob).addRootXpub(keyBob.xpub).setQuorum(1).create(client);
    new Asset.Builder().setAlias(aliasGold).addRootXpub(keyGold.xpub).setQuorum(1).create(client);
    new Asset.Builder()
        .setAlias(aliasSilver)
        .addRootXpub(keySilver.xpub)
        .setQuorum(1)
        .create(client);
    new Asset.Builder()
        .setAlias(aliasBronze)
        .addRootXpub(keyBronze.xpub)
        .setQuorum(1)
        .create(client);

    // Run a transaction to set up all of the UTXOs. Make lots of UTXOs to
    // avoid reservation errors later.
    Transaction.Builder builder = new Transaction.Builder();
    for (int i = 0; i < 10; i++) {
      builder
          .addAction(new Transaction.Action.Issue().setAssetAlias(aliasGold).setAmount(100))
          .addAction(new Transaction.Action.Issue().setAssetAlias(aliasSilver).setAmount(100))
          .addAction(new Transaction.Action.Issue().setAssetAlias(aliasBronze).setAmount(100))
          .addAction(
              new Transaction.Action.ControlWithAccount()
                  .setAccountAlias(aliasAlice)
                  .setAssetAlias(aliasGold)
                  .setAmount(100))
          .addAction(
              new Transaction.Action.ControlWithAccount()
                  .setAccountAlias(aliasBob)
                  .setAssetAlias(aliasSilver)
                  .setAmount(100))
          .addAction(
              new Transaction.Action.ControlWithAccount()
                  .setAccountAlias(aliasAlice)
                  .setAssetAlias(aliasBronze)
                  .setAmount(100));
    }
    Transaction.Template issuance = builder.build(client);
    Transaction.SubmitResponse resp = Transaction.submit(client, HsmSigner.sign(issuance));
    Transaction.Items txs =
        new Transaction.QueryBuilder()
            .setFilter("id=$1")
            .addFilterParameter(resp.id)
            .execute(client);
    issuanceTx = txs.next();

    // Grab the individual outputs so that we can spend them directly.
    for (Transaction.Output out : issuanceTx.outputs) {
      if (aliasAlice.equals(out.accountAlias) && aliasGold.equals(out.assetAlias)) {
        outputsAliceGold.add(out);
      } else if (aliasBob.equals(out.accountAlias) && aliasSilver.equals(out.assetAlias)) {
        outputsBobSilver.add(out);
      } else if (aliasAlice.equals(out.accountAlias) && aliasBronze.equals(out.assetAlias)) {
        outputsAliceBronze.add(out);
      }
    }
  }

  public void testSubmitUnbalanced() throws Exception {
    System.out.println("testSubmitUnbalanced");
    try {
      Transaction.Template tmpl =
          new Transaction.Builder()
              .addAction(
                  new Transaction.Action.SpendAccountUnspentOutput()
                      .setOutputId(outputsAliceGold.poll().id))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasBob)
                      .setAssetAlias(aliasGold)
                      .setAmount(11))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasAlice)
                      .setAssetAlias(aliasGold)
                      .setAmount(90))
              .build(client);
      Transaction.submit(client, HsmSigner.sign(tmpl));
    } catch (APIException ex) {
      if ("CH735".equals(ex.code)) {
        return;
      } else {
        throw ex;
      }
    }
    throw new Exception("expecting CH735 APIException");
  }

  public void testSubmitUnbalancedTwoAssets() throws Exception {
    System.out.println("testSubmitUnbalancedTwoAssets");
    try {
      Transaction.Template tmpl =
          new Transaction.Builder()
              .addAction(
                  new Transaction.Action.SpendAccountUnspentOutput()
                      .setOutputId(outputsAliceGold.poll().id))
              .addAction(
                  new Transaction.Action.SpendAccountUnspentOutput()
                      .setOutputId(outputsBobSilver.poll().id))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasBob)
                      .setAssetAlias(aliasGold)
                      .setAmount(10))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasAlice)
                      .setAssetAlias(aliasGold)
                      .setAmount(95))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasAlice)
                      .setAssetAlias(aliasSilver)
                      .setAmount(5))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasBob)
                      .setAssetAlias(aliasSilver)
                      .setAmount(90))
              .build(client);
      Transaction.submit(client, HsmSigner.sign(tmpl));
    } catch (APIException ex) {
      if ("CH735".equals(ex.code)) {
        return;
      } else {
        throw ex;
      }
    }
    throw new Exception("expecting CH735 APIException");
  }

  public void testSubmitUnsignedSpend() throws Exception {
    System.out.println("testSubmitUnsignedSpend");
    try {
      Transaction.Template tmpl =
          new Transaction.Builder()
              .addAction(
                  new Transaction.Action.SpendAccountUnspentOutput()
                      .setOutputId(outputsAliceGold.poll().id))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasBob)
                      .setAssetAlias(aliasGold)
                      .setAmount(10))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasAlice)
                      .setAssetAlias(aliasGold)
                      .setAmount(90))
              .build(client);
      Transaction.submit(client, tmpl);
    } catch (APIException ex) {
      if ("CH738".equals(ex.code)) {
        return;
      } else {
        throw ex;
      }
    }
    throw new Exception("expecting CH738 APIException");
  }

  public void testSubmitUnsignedIssuance() throws Exception {
    System.out.println("testSubmitUnsignedIssuance");
    try {
      Transaction.Template tmpl =
          new Transaction.Builder()
              .addAction(new Transaction.Action.Issue().setAssetAlias(aliasBronze).setAmount(100))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasBob)
                      .setAssetAlias(aliasBronze)
                      .setAmount(100))
              .build(client);
      Transaction.submit(client, tmpl);
    } catch (APIException ex) {
      if ("CH738".equals(ex.code)) {
        return;
      } else {
        throw ex;
      }
    }
    throw new Exception("expecting CH738 APIException");
  }

  public void testSubmitPartiallySigned() throws Exception {
    System.out.println("testSubmitPartiallySigned");
    try {
      Transaction.Template tmpl =
          new Transaction.Builder()
              .addAction(
                  new Transaction.Action.SpendAccountUnspentOutput()
                      .setOutputId(outputsAliceGold.poll().id))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountAlias(aliasBob)
                      .setAssetAlias(aliasGold)
                      .setAmount(101))
              .build(client);
      // Sign the current, unbalanced transaction
      tmpl = HsmSigner.sign(tmpl);
      // Add an issuance to make the transaction balanced, but don't sign it.
      tmpl =
          new Transaction.Builder(tmpl.rawTransaction)
              .addAction(new Transaction.Action.Issue().setAssetAlias(aliasGold).setAmount(1))
              .build(client);
      Transaction.submit(client, tmpl);
    } catch (APIException ex) {
      if ("CH738".equals(ex.code)) {
        return;
      } else {
        throw ex;
      }
    }
    throw new Exception("expecting CH738 APIException");
  }
}
