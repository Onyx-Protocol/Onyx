import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class UnspentOutputs {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    MockHsm.Key key = MockHsm.Key.create(context);
    HsmSigner.addKey(key, MockHsm.getSignerContext(context));

    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    Transaction.SubmitResponse issuanceTx = Transaction.submit(
      context,
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
          ).build(context)
      )
    );

    // snippet alice-unspent-outputs
    UnspentOutput.Items aliceUnspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("alice")
      .execute(context);

    while (aliceUnspentOutputs.hasNext()) {
      UnspentOutput utxo = aliceUnspentOutputs.next();
      System.out.println("Unspent output in alice account: " + utxo.transactionId + ":" + utxo.position);
    }
    // endsnippet

    // snippet gold-unspent-outputs
    UnspentOutput.Items goldUnspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("asset_alias=$1")
      .addFilterParameter("gold")
      .execute(context);

    while (goldUnspentOutputs.hasNext()) {
      UnspentOutput utxo = goldUnspentOutputs.next();
      System.out.println("Unspent output containing gold: " + utxo.transactionId + ":" + utxo.position);
    }
    // endsnippet

    String prevTransactionId = issuanceTx.id;

    // snippet build-transaction-all
    Transaction.Template spendOutput = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendAccountUnspentOutput()
        .setTransactionId(prevTransactionId)
        .setPosition(0)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(context);
    // endsnippet

    Transaction.submit(context, HsmSigner.sign(spendOutput));

    // snippet build-transaction-partial
    Transaction.Template spendOutputWithChange = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendAccountUnspentOutput()
        .setTransactionId(prevTransactionId)
        .setPosition(1)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(40)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(60)
      ).build(context);
    // endsnippet

    Transaction.submit(context, HsmSigner.sign(spendOutputWithChange));
  }
}
