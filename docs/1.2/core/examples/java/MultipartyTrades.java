import java.util.*;
import java.net.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class MultipartyTrades {
  public static void main(String[] args) throws Exception {
    // This demo is written to run on either one or two cores. Simply provide
    // different URLs to the following clients for the two-core version.
    Client aliceCore = new Client();
    Client bobCore = new Client();

    MockHsm.Key aliceDollarKey = MockHsm.Key.create(aliceCore);
    HsmSigner.addKey(aliceDollarKey, MockHsm.getSignerClient(aliceCore));

    MockHsm.Key bobBuckKey = MockHsm.Key.create(bobCore);
    HsmSigner.addKey(bobBuckKey, MockHsm.getSignerClient(bobCore));

    MockHsm.Key aliceKey = MockHsm.Key.create(aliceCore);
    HsmSigner.addKey(aliceKey, MockHsm.getSignerClient(aliceCore));

    MockHsm.Key bobKey = MockHsm.Key.create(bobCore);
    HsmSigner.addKey(bobKey, MockHsm.getSignerClient(bobCore));

    Asset aliceDollar = new Asset.Builder()
      .setAlias("aliceDollar")
      .addRootXpub(aliceDollarKey.xpub)
      .setQuorum(1)
      .create(aliceCore);

    Asset bobBuck = new Asset.Builder()
      .setAlias("bobBuck")
      .addRootXpub(bobBuckKey.xpub)
      .setQuorum(1)
      .create(bobCore);

    Account alice = new Account.Builder()
      .setAlias("alice")
      .addRootXpub(aliceKey.xpub)
      .setQuorum(1)
      .create(aliceCore);

    Account bob = new Account.Builder()
      .setAlias("bob")
      .addRootXpub(bobKey.xpub)
      .setQuorum(1)
      .create(bobCore);

    Transaction.submit(aliceCore, HsmSigner.sign(new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("aliceDollar")
        .setAmount(1000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("aliceDollar")
        .setAmount(1000)
      ).build(aliceCore)
    ));

    Transaction.submit(bobCore, HsmSigner.sign(new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("bobBuck")
        .setAmount(1000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("bobBuck")
        .setAmount(1000)
      ).build(bobCore)
    ));

    if (aliceCore.equals(bobCore)) {
      sameCore(aliceCore);
    }

    crossCore(aliceCore, bobCore, alice, bob, aliceDollar.id, bobBuck.id);
  }

  public static void sameCore(Client client) throws Exception {
    // snippet same-core-trade
    Transaction.Template trade = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("aliceDollar")
        .setAmount(50)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("bobBuck")
        .setAmount(100)
      ).addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("bobBuck")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("aliceDollar")
        .setAmount(50)
      ).build(client);

    Transaction.submit(client, HsmSigner.sign(trade));
    // endsnippet
  }

  public static void crossCore(
    Client aliceCore, Client bobCore,
    Account alice, Account bob,
    String aliceDollarAssetId, String bobBuckAssetId
  ) throws Exception {
    // snippet build-trade-alice
    Transaction.Template aliceTrade = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("aliceDollar")
        .setAmount(50)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetId(bobBuckAssetId)
        .setAmount(100)
      ).build(aliceCore);
    // endsnippet

    // snippet sign-trade-alice
    Transaction.Template aliceTradeSigned = HsmSigner.sign(aliceTrade.allowAdditionalActions());
    // endsnippet

    // snippet base-transaction-alice
    String baseTransactionFromAlice = aliceTradeSigned.rawTransaction;
    // endsnippet

    // snippet build-trade-bob
    Transaction.Template bobTrade = new Transaction.Builder()
      .setBaseTransaction(baseTransactionFromAlice)
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("bobBuck")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetId(aliceDollarAssetId)
        .setAmount(50)
      ).build(bobCore);
    // endsnippet

    // snippet sign-trade-bob
    Transaction.Template bobTradeSigned = HsmSigner.sign(bobTrade);
    // endsnippet

    // snippet submit-trade-bob
    Transaction.submit(bobCore, bobTradeSigned);
    // endsnippet
  }
}
