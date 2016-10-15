import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Balances {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    MockHsm.Key assetKey = MockHsm.Key.create(context);
    HsmSigner.addKey(assetKey);

    MockHsm.Key aliceKey = MockHsm.Key.create(context);
    HsmSigner.addKey(aliceKey);

    MockHsm.Key bobKey = MockHsm.Key.create(context);
    HsmSigner.addKey(bobKey);

    // snippet account-balance
    Balance.Items bank1Balances = new Balance.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("bank1")
      .execute(context);
    // endsnippet

    // snippet usd-iou-circulation
    Balance.Items bank1UsdCirculation = new Balance.QueryBuilder()
      .setFilter("asset_id=$1")
      .addFilterParameter("bank1_usd_iou")
      .execute(context);
    // endsnippet

    // snippet account-balance-sum-by-currency
    Balance.Items bank1CurrencyBalances = new Balance.QueryBuilder()
      .setFilter("account_alias=$1")
      .addFilterParameter("bank1")
      .addSumByParameter("asset_definition.currency")
      .execute(context);
    // endsnippet
  }
}
