import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Transactions {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    // snippet list-alice-transactions
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("inputs(account_alias=$1) AND outputs(account_alias=$1)")
      .addFilterParameter("alice")
      .execute(context);
    // endsnippet

    // snippet list-local-transactions
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("is_local=$1")
      .addFilterParameter("yes")
      .execute(context);
    // endsnippet

    // snippet create-feed
    new Transaction.Feed.Builder()
      .setAlias("alice_feed")
      .setFilter("account_alias=$1")
      .addFilterParameter("alice")
    // endsnippet

    // snippet process-feed
    Transaction.Feed feed = Transaction.Feed.getByAlias(context, 'alice_feed');
    while (true) {
      Transaction tx = feed.next(context);

      // process the tx...

      feed.ack(context);
    }
    // endsnippet

    // snippet issue-within-core
    Transaction.Template issuanceTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(1000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(1000)
      ).build(context);

    Transaction.Template signedIssuanceTransaction = HsmSigner.sign(issuanceTransaction));

    Transaction.submit(context, signedIssuanceTransaction);
    // endsnippet

    // snippet create-bob-issue-program
    ControlProgram bobProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(context);
    // endsnippet

    // snippet issue-to-bob-program
    Transaction.Template issuanceTransaction2 = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.program)
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);

    Transaction.Template signedIssuanceTransaction2 = HsmSigner.sign(issuanceTransaction2));

    Transaction.submit(context, signedIssuanceTransaction2);
    // endsnippet

    // snippet pay-within-core
    Transaction.Template simplePaymentTransaction1 = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);

    Transaction.Template signedSimplePaymentTransaction1 = HsmSigner.sign(simplePaymentTransaction1);

    Transaction.submit(context, signedSimplePaymentTransaction1);
    // endsnippet

    // snippet create-bob-payment-program
    ControlProgram bobProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(context);
    // endsnippet

    // snippet pay-between-cores
    Transaction.Template simplePaymentTransaction2 = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.program)
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);

    Transaction.Template signedSimplePaymentTransaction2 = HsmSigner.sign(simplePaymentTransaction2);

    Transaction.submit(context, signedSimplePaymentTransaction2);
    // endsnippet

    // snippet multiasset-within-core
    Transaction.Template multiAssetPaymentTransaction = new Transaction.Builder()
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
      ).build(context);

    Transaction.Template signedMultiAssetPaymentTransaction = HsmSigner.sign(multiAssetPaymentTransaction));

    Transaction.submit(context, signedMultiAssetPaymentTransaction);
    // endsnippet

    // snippet create-bob-multiasset-program
    ControlProgram bobProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(context);
    // endsnippet

    // snippet multiasset-between-cores
    Transaction.Template multiAssetPaymentTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("silver")
        .setAmount(20)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.program)
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram.program)
        .setAssetAlias("silver")
        .setAmount(20)
      ).build(context);

    Transaction.Template signedMultiAssetPaymentTransaction = HsmSigner.sign(multiAssetPaymentTransaction));

    Transaction.submit(context, signedMultiAssetPaymentTransaction);
    // endsnippet

    // snippet trade-within-core
    Transaction.Template assetTradeTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("silver")
        .setAmount(20)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("silver")
        .setAmount(20)
      ).build(context);

    Transaction.Template signedAssetTradeTransaction = HsmSigner.sign(assetTradeTransaction));

    Transaction.submit(context, signedAssetTradeTransaction);
    // endsnippet

    // snippet build-trade-a
    Transaction.Template tradeProposal = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("silver")
        .setAmount(20)
      ).build(context);
    // endsnippet

    // snippet sign-trade-a
    Transaction.Template signedTradeProposal = HsmSigner.sign(tradeProposal);
    // endsnippet

    // snippet build-trade-b
    Transaction.Template tradeTransaction = new Transaction.Builder()
      .setRawTransaction(signedTradeProposal.rawTransaction)
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("silver")
        .setAmount(20)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);
    // endsnippet

    // snippet sign-trade-b
    Transaction.Template signedTradeTransaction = HsmSigner.sign(tradeTransaction);
    // endsnippet

    // snippet submit-trade
    Transaction.submit(context, signedTradeTransaction);
    // endsnippet

    // snippet retire
    Transaction.Template retirementTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(50)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("gold")
        .setAmount(50)
      ).build(context);

    Transaction.Template signedRetirementTransaction = HsmSigner.sign(retirementTransaction));

    Transaction.submit(context, signedRetirementTransaction);
    // endsnippet
  }
}
