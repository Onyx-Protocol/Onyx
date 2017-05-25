import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class TransactionBasics {
  public static void main(String[] args) throws Exception {
    Client client = new Client();
    Client otherCoreClient = new Client();
    setup(client, otherCoreClient);

    // snippet issue-within-core
    Transaction.Template issuance = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(1000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(1000)
      ).build(client);

    Transaction.Template signedIssuance = HsmSigner.sign(issuance);

    Transaction.submit(client, signedIssuance);
    // endsnippet

    // snippet create-bob-issue-receiver
    Receiver bobIssuanceReceiver = new Account.ReceiverBuilder()
      .setAccountAlias("bob")
      .create(otherCoreClient);
    String bobIssuanceReceiverSerialized = bobIssuanceReceiver.toJson();
    // endsnippet

    // snippet issue-to-bob-receiver
    Transaction.Template issuanceToReceiver = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithReceiver()
        .setReceiver(Receiver.fromJson(bobIssuanceReceiverSerialized))
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(client);

    Transaction.Template signedIssuanceToReceiver = HsmSigner.sign(issuanceToReceiver);

    Transaction.submit(client, signedIssuanceToReceiver);
    // endsnippet

    // snippet pay-within-core
    Transaction.Template payment = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(client);

    Transaction.Template signedPayment = HsmSigner.sign(payment);

    Transaction.submit(client, signedPayment);
    // endsnippet

    // snippet create-bob-payment-receiver
    Receiver bobPaymentReceiver = new Account.ReceiverBuilder()
      .setAccountAlias("bob")
      .create(otherCoreClient);
    String bobPaymentReceiverSerialized = bobPaymentReceiver.toJson();
    // endsnippet

    // snippet pay-between-cores
    Transaction.Template paymentToReceiver = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithReceiver()
        .setReceiver(Receiver.fromJson(bobPaymentReceiverSerialized))
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(client);

    Transaction.Template signedPaymentToReceiver = HsmSigner.sign(paymentToReceiver);

    Transaction.submit(client, signedPaymentToReceiver);
    // endsnippet

    if (client.equals(otherCoreClient)) {
      // snippet multiasset-within-core
      Transaction.Template multiAssetPayment = new Transaction.Builder()
        .addAction(new Transaction.Action.SpendFromAccount()
          .setAccountAlias("alice")
          .setAssetAlias("gold")
          .setAmount(10)
        ).addAction(new Transaction.Action.SpendFromAccount()
          .setAccountAlias("alice")
          .setAssetAlias("silver")
          .setAmount(20)
        ).addAction(new Transaction.Action.ControlWithAccount()
          .setAccountAlias("bob")
          .setAssetAlias("gold")
          .setAmount(10)
        ).addAction(new Transaction.Action.ControlWithAccount()
          .setAccountAlias("bob")
          .setAssetAlias("silver")
          .setAmount(20)
        ).build(client);

      Transaction.Template signedMultiAssetPayment = HsmSigner.sign(multiAssetPayment);

      Transaction.submit(client, signedMultiAssetPayment);
      // endsnippet
    }

    // snippet create-bob-multiasset-receiver
    Receiver bobGoldReceiver = new Account.ReceiverBuilder()
      .setAccountAlias("bob")
      .create(otherCoreClient);
    String bobGoldReceiverSerialized = bobGoldReceiver.toJson();

    Receiver bobSilverReceiver = new Account.ReceiverBuilder()
      .setAccountAlias("bob")
      .create(otherCoreClient);
    String bobSilverReceiverSerialized = bobSilverReceiver.toJson();
    // endsnippet

    // snippet multiasset-between-cores
    Transaction.Template multiAssetToReceiver = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("silver")
        .setAmount(20)
      ).addAction(new Transaction.Action.ControlWithReceiver()
        .setReceiver(Receiver.fromJson(bobGoldReceiverSerialized))
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithReceiver()
        .setReceiver(Receiver.fromJson(bobSilverReceiverSerialized))
        .setAssetAlias("silver")
        .setAmount(20)
      ).build(client);

    Transaction.Template signedMultiAssetToReceiver = HsmSigner.sign(multiAssetToReceiver);

    Transaction.submit(client, signedMultiAssetToReceiver);
    // endsnippet

    // snippet retire
    Transaction.Template retirement = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(50)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("gold")
        .setAmount(50)
      ).build(client);

    Transaction.Template signedRetirement = HsmSigner.sign(retirement);

    Transaction.submit(client, signedRetirement);
    // endsnippet
  }

  public static void setup(Client client, Client otherCoreClient) throws Exception {
    MockHsm.Key aliceKey = MockHsm.Key.create(client);
    HsmSigner.addKey(aliceKey, MockHsm.getSignerClient(client));

    MockHsm.Key bobKey = MockHsm.Key.create(otherCoreClient);
    HsmSigner.addKey(bobKey, MockHsm.getSignerClient(otherCoreClient));

    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(aliceKey.xpub)
      .setQuorum(1)
      .create(client);

    new Asset.Builder()
      .setAlias("silver")
      .addRootXpub(aliceKey.xpub)
      .setQuorum(1)
      .create(client);

    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(aliceKey.xpub)
      .setQuorum(1)
      .create(client);

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(bobKey.xpub)
      .setQuorum(1)
      .create(otherCoreClient);

    Transaction.submit(client, HsmSigner.sign(new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("silver")
        .setAmount(1000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("silver")
        .setAmount(1000)
      ).build(client)
    ));
  }
}
