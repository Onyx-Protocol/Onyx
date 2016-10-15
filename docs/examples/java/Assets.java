import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class Assets {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    MockHsm.Key assetKey = MockHsm.Key.create(context);
    HsmSigner.addKey(assetKey);

    MockHsm.Key aliceKey = MockHsm.Key.create(context);
    HsmSigner.addKey(aliceKey);

    MockHsm.Key bobKey = MockHsm.Key.create(context);
    HsmSigner.addKey(bobKey);

    // snippet create-asset-acme-common
    // Create the asset definition
    Map<String, Object> acmeCommonDef = new HashMap<>();
    acmeCommonDef.put("issuer", "Acme Inc.");
    acmeCommonDef.put("type", "security");
    acmeCommonDef.put("subtype", "private");
    acmeCommonDef.put("class", "common");

    // Build the asset
    new Asset.Builder()
      .setAlias("acme_common")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addTag("internal_rating", "1")
      .setDefinition(acmeCommonDef)
      .create(context);
    // endsnippet

    // snippet create-asset-acme-preferred
    // Create the asset definition
    Map<String, Object> acmePreferredDef = new HashMap<>();
    acmePreferredDef.put("issuer", "Acme Inc.");
    acmePreferredDef.put("type", "security");
    acmePreferredDef.put("subtype", "private");
    acmePreferredDef.put("class", "perferred");

    // Build the asset
    new Asset.Builder()
      .setAlias("acme_preferred")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .addTag("internal_rating", "2")
      .setDefinition(acmePreferredDef)
      .create(context);
    // endsnippet

    // snippet list-local-assets
    Asset.Items localAssets = new Asset.QueryBuilder()
      .setFilter("is_local=$1")
      .addFilterParameter("yes")
      .execute(context);
    // endsnippet

    // snippet list-private-preferred-securities
    Assets.Items common = new Asset.QueryBuilder()
      .setFilter("definition.type=$1 AND definition.subtype=$2 AND definition.class=$3")
      .addFilterParameter("security")
      .addFilterParameter("private")
      .addFilterParameter("preferred")
      .execute(context);
    // endsnippet

    // snippet build-issue
    Transaction.Template issuanceTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme_common")
        .setAmount(1000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("acme_treasury")
        .setAssetAlias("acme_common")
        .setAmount(1000)
      ).build(context);
    // endsnippet

    // snippet sign-issue
    Transaction.Template signedIssuanceTransaction = HsmSigner.sign(issuanceTransaction);
    // endsnippet

    // snippet submit-issue
    Transaction.submit(context, signedIssuanceTransaction);
    // endsnippet

    // snippet external-issue
    Transaction.Template externalIssuance = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme_common")
        .setAmount(2000)
      ).addAction(new Transaction.Action.ControlWithProgram()
        .setControlProgram(externalProgram)
        .setAssetAlias("acme_common")
        .setAmount(2000)
      ).build(context);

    Transaction.submit(context, HsmSigner.sign(externalIssuance));
    // endsnippet

    // snippet build-trade-a
    Transaction.Template tradeProposal = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme_common")
        .setAmount(1000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("acme_treasury")
        .setAssetAlias("USD")
        .setAmount(5000000)
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
        .setAssetAlias("USD")
        .setAmount(5000000)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("acme_commmon")
        .setAmount(1000)
      ).build(context);
    // endsnippet

    // snippet sign-trade-b
    Transaction.Template signedTradeTransaction = HsmSigner.sign(tradeTransaction);
    // endsnippet

    // snippet submit-trade
    Transaction.submit(context, signedTradeTransaction);
    // endsnippet

    // snippet build-retire
    Transaction.Template retirementTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("acme_treasury")
        .setAssetAlias("acme_common")
        .setAmount(50)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("acme_common")
        .setAmount(50)
      ).build(context);
    // endsnippet

    // snippet sign-retire
    Transaction.Template signedRetirementTransaction = HsmSigner.sign(retirementTransaction);
    // endsnippet

    // snippet submit-retire
    Transaction.submit(context, signedRetirementTransaction);
    // endsnippet

    // snippet list-issuances
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("inputs(action=$1 AND asset_alias=$2)")
      .addFilterParameter("issue")
      .addFilterParameter("acme_common")
      .execute(context);
    // endsnippet

    // snippet list-transfers
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("inputs(action=$1 AND asset_alias=$2)")
      .addFilterParameter("spend")
      .addFilterParameter("acme_common")
      .execute(context);
    // endsnippet

    // snippet list-retirements
    Transaction.Items transactions = new Transaction.QueryBuilder()
      .setFilter("outputs(action=$1 AND asset_alias=$2)")
      .addFilterParameter("retire")
      .addFilterParameter("acme_common")
      .execute(context);
    // endsnippet

    // snippet list-acme-common-balance
    Balance.Items balances = new Balance.QueryBuilder()
      .setFilter("asset_alias=$1")
      .addFilterParameter("acme_common")
      .execute(context);
    // endsnippet

    // snippet list-acme-balance
    Balance.Items balances = new Balance.QueryBuilder()
      .setFilter("asset_definition.entity=$1")
      .addFilterParameter("Acme Inc.")
      .execute(context);
    // endsnippet

    // snippet list-acme-common-unspents
    UnspentOutput.Items acmeCommonUnspentOutputs = new UnspentOutput.QueryBuilder()
      .setFilter("asset_alias='$1'")
      .addFilterParameter("acme_common")
      .execute(context);
    // endsnippet
  }
}
