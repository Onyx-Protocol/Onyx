import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Balances {
  public static void main(String[] args) throws Exception {
    Client client = new Client();
    setup(client);

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
