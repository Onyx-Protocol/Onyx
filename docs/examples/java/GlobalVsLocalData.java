import java.util.*;

import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class GlobalVsLocalData {
  public static void main(String[] args) throws Exception {
    Context context = new Context();

    // snippet create-accounts-with-tags
    new Account.Builder()
      .setAlias("alice")
      .addXpub(mainKey.xpub)
      .setQuorum(1)
      .addTag("type", "checking")
      .addTag("first_name", "Alice")
      .addTag("last_name", "Jones")
      .addTag("user_id", "12345")
      .addTag("status", "enabled")
      .create(context);

    new Account.Builder()
      .setAlias("bob")
      .addXpub(mainKey.xpub)
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
      .setAlias("acme-bond")
      .addXpub(mainKey.xpub)
      .setQuorum(1)
      .addTag("internal_rating", "B")
      .addDefinitionField("type", "security");
      .addDefinitionField("sub-type", "corporate-bond");
      .addDefinitionField("entity", "Acme Inc.");
      .addDefinitionField("maturity", "2016-09-01T18:24:47+00:00");
      .create(context);
    // endsnippet

    // snippet build-tx-with-tx-ref-data
    Transaction.Template issuanceTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme-bond")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("acme-bond")
        .setAmount(100)
      ).addAction(new Transaction.Action.SetTransactionReferenceData()
        .addReferenceDataField("external_reference", "12345");
      ).build(context);
    // endsnippet

    // snippet build-tx-with-action-ref-data
    Transaction.Template issuanceTransaction = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("acme-bond")
        .setAmount(100)
      ).addAction(new Transaction.Action.Retire()
        .setAssetAlias("acme-bond")
        .setAmount(100)
        .addReferenceDataField("external_reference", "12345");
      ).build(context);
    // endsnippet
  }
}
