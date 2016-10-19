import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Accounts {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    MockHsm.Key assetKey = MockHsm.Key.create(context);
    HsmSigner.addKey(assetKey, MockHsm.getSignerContext(context));

    MockHsm.Key aliceKey = MockHsm.Key.create(context);
    HsmSigner.addKey(aliceKey, MockHsm.getSignerContext(context));

    MockHsm.Key bobKey = MockHsm.Key.create(context);
    HsmSigner.addKey(bobKey, MockHsm.getSignerContext(context));

    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(assetKey.xpub)
      .setQuorum(1)
      .create(context);

    new Asset.Builder()
      .setAlias("silver")
      .addRootXpub(assetKey.xpub)
      .setQuorum(1)
      .create(context);

    // snippet create-account-alice
    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(aliceKey.xpub)
      .setQuorum(1)
      .addTag("type", "checking")
      .addTag("first_name", "Alice")
      .addTag("last_name", "Jones")
      .addTag("user_id", "12345")
      .create(context);
    // endsnippet

    // snippet create-account-bob
    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(bobKey.xpub)
      .setQuorum(1)
      .addTag("type", "savings")
      .addTag("first_name", "Bob")
      .addTag("last_name", "Smith")
      .addTag("user_id", "67890")
      .create(context);
    // endsnippet

    // snippet list-accounts-by-tag
    Account.Items accounts = new Account.QueryBuilder()
      .setFilter("tags.type=$1")
      .addFilterParameter("savings")
      .execute(context);

    while (accounts.hasNext()) {
      Account a = accounts.next();
      System.out.println("Account ID " + a.id + ", alias " + a.alias);
    }
    // endsnippet

    Transaction.Template fundAliceTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(context);

    Transaction.submit(context, HsmSigner.sign(fundAliceTransaction));

    Transaction.Template fundBobTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("silver")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("silver")
        .setAmount(100)
      ).build(context);

    Transaction.submit(context, HsmSigner.sign(fundBobTransaction));

    // snippet build-transfer
    Transaction.Template spendingTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);
    // endsnippet

    // snippet sign-transfer
    Transaction.Template signedSpendingTransaction = HsmSigner.sign(spendingTransaction);
    // endsnippet

    // snippet submit-transfer
    Transaction.submit(context, signedSpendingTransaction);
    // endsnippet

    // snippet create-control-program
    ControlProgram bobProgram = new ControlProgram.Builder()
      .controlWithAccountByAlias("bob")
      .create(context);
    // endsnippet

    // snippet transfer-to-control-program
    Transaction.Template spendingTransaction2 = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(bobProgram)
        .setAssetAlias("gold")
        .setAmount(10)
      ).build(context);

    Transaction.submit(context, HsmSigner.sign(spendingTransaction2));
    // endsnippet

    // snippet list-account-txs
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("inputs(account_alias=$1) AND outputs(account_alias=$1)")
      .addFilterParameter("alice")
      .execute(context);

    while (transactions.hasNext()) {
      Transaction t = transactions.next();
      System.out.println("Transaction " + t.id + " at " + t.timestamp);
    }
    // endsnippet

    // snippet list-account-balances
    Balance.Items balances = new Balance.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("alice")
      .execute(context);

    while (balances.hasNext()) {
      Balance b = balances.next();
      System.out.println(
        "Alice's balance of " + b.sumBy.get("asset_alias") +
        ": " + b.amount
      );
    }
    // endsnippet

    // snippet list-account-unspent-outputs
    UnspentOutput.Items unspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("account_alias=$1 AND asset_alias=$2")
      .addFilterParameter("alice")
      .addFilterParameter("gold")
      .execute(context);

    while (unspentOutputs.hasNext()) {
      UnspentOutput u = unspentOutputs.next();
      System.out.println("Transaction " + u.transactionId + " position " + u.position);
    }
    // endsnippet
  }
}
