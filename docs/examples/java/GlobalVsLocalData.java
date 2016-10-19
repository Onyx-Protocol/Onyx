import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class GlobalVsLocalData {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    MockHsm.Key assetKey = MockHsm.Key.create(context);
    HsmSigner.addKey(assetKey, MockHsm.getSignerContext(context));

    MockHsm.Key aliceKey = MockHsm.Key.create(context);
    HsmSigner.addKey(aliceKey, MockHsm.getSignerContext(context));

    MockHsm.Key bobKey = MockHsm.Key.create(context);
    HsmSigner.addKey(bobKey, MockHsm.getSignerContext(context));

    // snippet create-accounts-with-tags
    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(aliceKey.xpub)
      .setQuorum(1)
      .addTag("type", "checking")
      .addTag("first_name", "Alice")
      .addTag("last_name", "Jones")
      .addTag("user_id", "12345")
      .addTag("status", "enabled")
      .create(context);

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(bobKey.xpub)
      .setQuorum(1)
      .addTag("type", "checking")
      .addTag("first_name", "Bob")
      .addTag("last_name", "Smith")
      .addTag("user_id", "67890")
      .addTag("status", "enabled")
      .create(context);
    // endsnippet

    // snippet create-asset-with-tags-and-definition
    new Asset.Builder()
      .setAlias("acme_bond")
      .addRootXpub(assetKey.xpub)
      .setQuorum(1)
      .addTag("internal_rating", "B")
      .addDefinitionField("type", "security")
      .addDefinitionField("sub-type", "corporate-bond")
      .addDefinitionField("entity", "Acme Inc.")
      .addDefinitionField("maturity", "2016-09-01T18:24:47+00:00")
      .create(context);
    // endsnippet

    // snippet build-tx-with-tx-ref-data
    Transaction.Template txWithRefData = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme_bond")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("acme_bond")
        .setAmount(100)
      ).addAction(new Transaction.Action.SetTransactionReferenceData()
        .addReferenceDataField("external_reference", "12345")
      ).build(context);
    // endsnippet

    Transaction.submit(context, HsmSigner.sign(txWithRefData));

    // snippet build-tx-with-action-ref-data
    Transaction.Template txWithActionRefData = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme_bond")
        .setAmount(100)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("acme_bond")
        .setAmount(100)
        .addReferenceDataField("external_reference", "12345")
      ).build(context);
    // endsnippet

    Transaction.submit(context, HsmSigner.sign(txWithActionRefData));
  }
}
