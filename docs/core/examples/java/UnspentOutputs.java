import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class UnspentOutputs {
  public static void main(String[] args) throws Exception {
    Client client = new Client();

    MockHsm.Key key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));

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

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    Transaction.SubmitResponse issuanceTx = Transaction.submit(
      client,
      HsmSigner.sign(
        new Transaction.Builder()
          .addAction(new Transaction.Action.Issue()
            .setAssetAlias("gold")
            .setAmount(200)
          ).addAction(new Transaction.Action.ControlWithAccount()
            .setAccountAlias("alice")
            .setAssetAlias("gold")
            .setAmount(100)
          ).addAction(new Transaction.Action.ControlWithAccount()
            .setAccountAlias("alice")
            .setAssetAlias("gold")
            .setAmount(100)
          ).build(client)
      )
    );

    // snippet alice-unspent-outputs
    UnspentOutput.Items aliceUnspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("alice")
      .execute(client);

    while (aliceUnspentOutputs.hasNext()) {
      UnspentOutput utxo = aliceUnspentOutputs.next();
      System.out.println("Unspent output in alice account: " + utxo.id);
    }
    // endsnippet

    // snippet gold-unspent-outputs
    UnspentOutput.Items goldUnspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("asset_alias=$1")
      .addFilterParameter("gold")
      .execute(client);

    while (goldUnspentOutputs.hasNext()) {
      UnspentOutput utxo = goldUnspentOutputs.next();
      System.out.println("Unspent output containing gold: " + utxo.id);
    }
    // endsnippet

    Transaction prevTransaction = new Transaction.QueryBuilder()
      .setFilter("id=$1")
      .addFilterParameter(issuanceTx.id)
      .execute(client)
      .next();

    // snippet build-transaction-all
    Transaction.Template spendOutput = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendAccountUnspentOutput()
        .setOutputId(prevTransaction.outputs.get(0).id)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(client);
    // endsnippet

    Transaction.submit(client, HsmSigner.sign(spendOutput));

    // snippet build-transaction-partial
    Transaction.Template spendOutputWithChange = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendAccountUnspentOutput()
        .setOutputId(prevTransaction.outputs.get(1).id)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(40)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(60)
      ).build(client);
    // endsnippet

    Transaction.submit(client, HsmSigner.sign(spendOutputWithChange));
  }
}
