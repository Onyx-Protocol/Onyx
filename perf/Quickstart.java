import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.http.Client;
import com.chain.signing.HsmSigner;

import java.math.BigInteger;
import java.net.URL;
import java.util.Arrays;

public class Quickstart {
  public static void main(String[] args) throws Exception {
    Client client = new Client(new URL(System.getenv("CHAIN_API_URL")));
    MockHsm.Key mainKey = MockHsm.Key.create(client);
    HsmSigner.addKey(mainKey, client);

    new Account.Builder().setAlias("alice").addRootXpub(mainKey.xpub).setQuorum(1).create(client);

    new Account.Builder().setAlias("bob").addRootXpub(mainKey.xpub).setQuorum(1).create(client);

    new Asset.Builder().setAlias("gold").addRootXpub(mainKey.xpub).setQuorum(1).create(client);

    Transaction.Template issuance =
        new Transaction.Builder()
            .addAction(new Transaction.Action.Issue().setAssetAlias("gold").setAmount(100))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias("alice")
                    .setAssetAlias("gold")
                    .setAmount(100))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(issuance));

    Transaction.Template spending =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias("alice")
                    .setAssetAlias("gold")
                    .setAmount(10))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAccountAlias("bob")
                    .setAssetAlias("gold")
                    .setAmount(10))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(spending));

    Transaction.Template retirement =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAccountAlias("bob")
                    .setAssetAlias("gold")
                    .setAmount(5))
            .addAction(new Transaction.Action.Retire().setAssetAlias("gold").setAmount(5))
            .build(client);
    Transaction.submit(client, HsmSigner.sign(retirement));
  }
}
