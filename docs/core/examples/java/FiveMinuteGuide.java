import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class FiveMinuteGuide {
  public static void main(String[] args) throws Exception {
    // snippet create-client
    Client client = new Client();
    // endsnippet

    // snippet create-key
    MockHsm.Key key = MockHsm.Key.create(client);
    // endsnippet

    // snippet signer-add-key
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    // endsnippet

    // snippet create-asset
    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);
    // endsnippet

    // snippet create-account-alice
    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);
    // endsnippet

    // snippet create-account-bob
    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);
    // endsnippet

    // snippet issue
    Transaction.Template issuance = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(client);

    Transaction.submit(client, HsmSigner.sign(issuance));
    // endsnippet

    // snippet spend
    Transaction.Template spending = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10))
      .addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(client);

    Transaction.submit(client, HsmSigner.sign(spending));
    // endsnippet

    // snippet retire
    Transaction.Template retirement = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(5)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("gold")
        .setAmount(5)
      ).build(client);

    Transaction.submit(client, HsmSigner.sign(retirement));
    // endsnippet
  }
}
