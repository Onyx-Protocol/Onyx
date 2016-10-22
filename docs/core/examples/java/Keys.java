import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Keys {
  public static void main(String[] args) throws Exception {
    Client client = new Client();

    // snippet create-key
    MockHsm.Key key = MockHsm.Key.create(client);
    // endsnippet

    // snippet signer-add-key
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));
    // endsnippet

    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    Transaction.Template unsigned = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(client);

    // snippet sign-transaction
    Transaction.Template signed = HsmSigner.sign(unsigned);
    // endsnippet
  }
}
