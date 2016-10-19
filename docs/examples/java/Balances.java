import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Balances {
  public static void main(String[] args) throws Exception {
    Context context = new Context();
    setup(context);

    // snippet account-balance
    Balance.Items bank1Balances = new Balance.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("bank1")
      .execute(context);

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
      .execute(context);

    Balance bank1UsdIouCirculation = bank1UsdIouBalances.next();
    System.out.println("Total circulation of Bank 1 USD IOU: " + bank1UsdIouCirculation.amount);
    // endsnippet

    // snippet account-balance-sum-by-currency
    Balance.Items bank1CurrencyBalances = new Balance.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("bank1")
      .setSumBy(Arrays.asList("asset_definition.currency"))
      .execute(context);

    while (bank1CurrencyBalances.hasNext()) {
      Balance b = bank1CurrencyBalances.next();
      System.out.println(
        "Bank 1 balance of " + b.sumBy.get("asset_definition.currency") +
        "-denominated currencies : " + b.amount
      );
    }
    // endsnippet
  }

  public static void setup(Context context) throws Exception {
    MockHsm.Key key = MockHsm.Key.create(context);
    HsmSigner.addKey(key, MockHsm.getSignerContext(context));

    new Asset.Builder()
      .setAlias("bank1_usd_iou")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addDefinitionField("currency", "USD")
      .create(context);

    new Asset.Builder()
      .setAlias("bank1_euro_iou")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addDefinitionField("currency", "Euro")
      .create(context);

    new Asset.Builder()
      .setAlias("bank2_usd_iou")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addDefinitionField("currency", "USD")
      .create(context);

    new Account.Builder()
      .setAlias("bank1")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    new Account.Builder()
      .setAlias("bank2")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(context);

    Transaction.submit(context, HsmSigner.sign(new Transaction.Builder()
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
      ).build(context)
    ));
  }
}
