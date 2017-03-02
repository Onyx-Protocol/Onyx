import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class GlobalVsLocalData {
  public static void main(String[] args) throws Exception {
    Client client = new Client();

    MockHsm.Key assetKey = MockHsm.Key.create(client);
    HsmSigner.addKey(assetKey, MockHsm.getSignerClient(client));

    MockHsm.Key aliceKey = MockHsm.Key.create(client);
    HsmSigner.addKey(aliceKey, MockHsm.getSignerClient(client));

    MockHsm.Key bobKey = MockHsm.Key.create(client);
    HsmSigner.addKey(bobKey, MockHsm.getSignerClient(client));

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
      .create(client);

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(bobKey.xpub)
      .setQuorum(1)
      .addTag("type", "checking")
      .addTag("first_name", "Bob")
      .addTag("last_name", "Smith")
      .addTag("user_id", "67890")
      .addTag("status", "enabled")
      .create(client);
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
      .create(client);
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
      ).build(client);
    // endsnippet

    Transaction.submit(client, HsmSigner.sign(txWithRefData));

    // snippet build-tx-with-action-ref-data
    Transaction.Template txWithActionRefData = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme_bond")
        .setAmount(100)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("acme_bond")
        .setAmount(100)
        .addReferenceDataField("external_reference", "12345")
      ).build(client);
    // endsnippet

    Transaction.submit(client, HsmSigner.sign(txWithActionRefData));
  }
}
