import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Queries {
  public static void main(String[] args) throws Exception {
    Client client = new Client();
    setup(client);

    // snippet list-alice-transactions
    Transaction.Items aliceTransactions = new Transaction.QueryBuilder()
      .setFilter("inputs(account_alias=$1) OR outputs(account_alias=$1)")
      .addFilterParameter("alice")
      .execute(client);

    while (aliceTransactions.hasNext()) {
      Transaction transaction = aliceTransactions.next();
      System.out.println("Alice's transaction " + transaction.id);
      for (Transaction.Input input: transaction.inputs) {
        if (input.accountAlias != null) {
          System.out.println("  -" + input.amount + " " + input.assetAlias);
        }
      }
      for (Transaction.Output output: transaction.outputs) {
        if (output.accountAlias != null) {
          System.out.println("  +" + output.amount + " " + output.assetAlias);
        }
      }
    }
    // endsnippet

    // snippet list-checking-transactions
    Transaction.Items checkingTransactions = new Transaction.QueryBuilder()
      .setFilter("inputs(account_tags.type=$1) OR outputs(account_tags.type=$1)")
      .addFilterParameter("checking")
      .execute(client);

    while (checkingTransactions.hasNext()) {
      Transaction transaction = checkingTransactions.next();
      System.out.println("Checking account transaction " + transaction.id);
      for (Transaction.Input input: transaction.inputs) {
        if (input.accountAlias != null) {
          System.out.println("  -" + input.amount + " " + input.assetAlias);
        }
      }
      for (Transaction.Output output: transaction.outputs) {
        if (output.accountAlias != null) {
          System.out.println("  +" + output.amount + " " + output.assetAlias);
        }
      }
    }
    // endsnippet

    // snippet list-local-transactions
    Transaction.Items localTransactions = new Transaction.QueryBuilder()
      .setFilter("is_local=$1")
      .addFilterParameter("yes")
      .execute(client);

    while (localTransactions.hasNext()) {
      Transaction transaction = localTransactions.next();
      System.out.println("Local transaction " + transaction.id);
    }
    // endsnippet

    // snippet list-local-assets
    Asset.Items localAssets = new Asset.QueryBuilder()
      .setFilter("is_local=$1")
      .addFilterParameter("yes")
      .execute(client);

    while (localAssets.hasNext()) {
      Asset asset = localAssets.next();
      System.out.println("Local asset " + asset.id + " (" + asset.alias + ")");
    }
    // endsnippet

    // snippet list-usd-assets
    Asset.Items usdAssets = new Asset.QueryBuilder()
      .setFilter("definition.currency=$1")
      .addFilterParameter("USD")
      .execute(client);

    while (usdAssets.hasNext()) {
      Asset asset = usdAssets.next();
      System.out.println("USD asset " + asset.id + " (" + asset.alias + ")");
    }
    // endsnippet

    // snippet list-checking-accounts
    Account.Items checkingAccounts = new Account.QueryBuilder()
      .setFilter("tags.type=$1")
      .addFilterParameter("checking")
      .execute(client);

    while (checkingAccounts.hasNext()) {
      Account account = checkingAccounts.next();
      System.out.println("Checking account " + account.id + " (" + account.alias + ")");
    }
    // endsnippet

    // snippet list-alice-unspents
    UnspentOutput.Items aliceUnspentOuputs = new UnspentOutput.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("alice")
      .execute(client);

    while (aliceUnspentOuputs.hasNext()) {
      UnspentOutput utxo = aliceUnspentOuputs.next();
      System.out.println("Alice's unspent output: " + utxo.amount + " " + utxo.assetAlias);
    }
    // endsnippet

    // snippet list-checking-unspents
    UnspentOutput.Items checkingUnspentOuputs = new UnspentOutput.QueryBuilder()
      .setFilter("account_tags.type=$1")
      .addFilterParameter("checking")
      .execute(client);

    while (checkingUnspentOuputs.hasNext()) {
      UnspentOutput utxo = checkingUnspentOuputs.next();
      System.out.println("Checking account unspent output: " + utxo.amount + " " + utxo.assetAlias);
    }
    // endsnippet

    // snippet account-balance
    Balance.Items bank1Balances = new Balance.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("bank1")
      .execute(client);

    while (bank1Balances.hasNext()) {
      Balance b = bank1Balances.next();
      System.out.println(
        "Bank 1 balance of " + b.sumBy.get("asset_alias") +
        ": " + b.amount
      );
    }
    // endsnippet

    // snippet checking-accounts-balance
    Balance.Items checkingBalances = new Balance.QueryBuilder()
      .setFilter("account_tags.type=$1")
      .addFilterParameter("checking")
      .execute(client);

    while (checkingBalances.hasNext()) {
      Balance b = checkingBalances.next();
      System.out.println(
        "Checking account balance of " + b.sumBy.get("asset_alias") +
        ": " + b.amount
      );
    }
    // endsnippet

    // snippet usd-iou-circulation
    Balance.Items bank1UsdIouBalances = new Balance.QueryBuilder()
      .setFilter("asset_alias=$1")
      .addFilterParameter("bank1_usd_iou")
      .execute(client);

    Balance bank1UsdIouCirculation = bank1UsdIouBalances.next();
    System.out.println("Total circulation of Bank 1 USD IOU: " + bank1UsdIouCirculation.amount);
    // endsnippet

    // snippet account-balance-sum-by-currency
    Balance.Items bank1CurrencyBalances = new Balance.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("bank1")
      .setSumBy(Arrays.asList("asset_definition.currency"))
      .execute(client);

    while (bank1CurrencyBalances.hasNext()) {
      Balance b = bank1CurrencyBalances.next();
      System.out.println(
        "Bank 1 balance of " + b.sumBy.get("asset_definition.currency") +
        "-denominated currencies : " + b.amount
      );
    }
    // endsnippet
  }

  public static void setup(Client client) throws Exception {
    MockHsm.Key key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));

    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    new Asset.Builder()
      .setAlias("silver")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    new Account.Builder()
      .setAlias("alice")
      .addTag("type", "checking")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    Transaction.submit(client, HsmSigner.sign(new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(1000))
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("silver")
        .setAmount(1000))
      .addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(1000))
      .addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("silver")
        .setAmount(1000))
      .build(client)));

    Transaction.submit(client, HsmSigner.sign(new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10))
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("bob")
        .setAssetAlias("silver")
        .setAmount(10))
      .addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("silver")
        .setAmount(10))
      .addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10))
      .build(client)));

    new Asset.Builder()
      .setAlias("bank1_usd_iou")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addDefinitionField("currency", "USD")
      .create(client);

    new Asset.Builder()
      .setAlias("bank1_euro_iou")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addDefinitionField("currency", "Euro")
      .create(client);

    new Asset.Builder()
      .setAlias("bank2_usd_iou")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addDefinitionField("currency", "USD")
      .create(client);

    new Account.Builder()
      .setAlias("bank1")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    new Account.Builder()
      .setAlias("bank2")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    Transaction.submit(client, HsmSigner.sign(new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("bank1_usd_iou")
        .setAmount(2000000)
      ).addAction(new Transaction.Action.Issue()
        .setAssetAlias("bank2_usd_iou")
        .setAmount(2000000)
      ).addAction(new Transaction.Action.Issue()
        .setAssetAlias("bank1_euro_iou")
        .setAmount(2000000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bank1")
        .setAssetAlias("bank1_usd_iou")
        .setAmount(1000000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bank1")
        .setAssetAlias("bank1_euro_iou")
        .setAmount(1000000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bank1")
        .setAssetAlias("bank2_usd_iou")
        .setAmount(1000000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bank2")
        .setAssetAlias("bank1_usd_iou")
        .setAmount(1000000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bank2")
        .setAssetAlias("bank1_euro_iou")
        .setAmount(1000000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bank2")
        .setAssetAlias("bank2_usd_iou")
        .setAmount(1000000)
      ).build(client)
    ));
  }
}
