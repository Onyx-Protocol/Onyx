import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class UnspentOutputs {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    // snippet alice-unspent-outputs
    UnspentOutput.Items aliceUnspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("alice")
      .execute(context);
    // endsnippet

    // snippet gold-unspent-outputs
    UnspentOutput.Items goldUnspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("asset_alias=$1")
      .addFilterParameter("gold")
      .execute(context);
    // endsnippet

    // snippet build-transaction-all
    Transaction.Template spend1 = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendAccountUnspentOutput()
        .setTransactionId("ad8e8aa37b0969ec60151674c821f819371152156782f107ed49724b8edd7b24")
        .setPosition(1)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(context);
    // endsnippet

    // snippet build-transaction-partial
    Transaction.Template spend1 = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendAccountUnspentOutput()
        .setTransactionId("ad8e8aa37b0969ec60151674c821f819371152156782f107ed49724b8edd7b24")
        .setPosition(1)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(40)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setPurpose("change")
        .setAmount(60)
      ).build(context);
    // endsnippet
  }
}
