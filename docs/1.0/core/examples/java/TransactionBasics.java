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

    // snippet create-bob-issue-program
    ControlProgram bobProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(otherCoreClient);
    // endsnippet

    // snippet issue-to-bob-program
    Transaction.Template issuanceToProgram = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.controlProgram)
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(client);

    Transaction.Template signedIssuanceToProgram = HsmSigner.sign(issuanceToProgram);

    Transaction.submit(client, signedIssuanceToProgram);
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

    // snippet create-bob-payment-program
    bobProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(otherCoreClient);
    // endsnippet

    // snippet pay-between-cores
    Transaction.Template paymentToProgram = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.controlProgram)
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(client);

    Transaction.Template signedPaymentToProgram = HsmSigner.sign(paymentToProgram);

    Transaction.submit(client, signedPaymentToProgram);
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

    // snippet create-bob-multiasset-program
    bobProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(otherCoreClient);
    // endsnippet

    // snippet multiasset-between-cores
    Transaction.Template multiAssetToProgram = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("silver")
        .setAmount(20)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.controlProgram)
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.controlProgram)
        .setAssetAlias("silver")
        .setAmount(20)
      ).build(client);

    Transaction.Template signedMultiAssetToProgram = HsmSigner.sign(multiAssetToProgram);

    Transaction.submit(client, signedMultiAssetToProgram);
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
