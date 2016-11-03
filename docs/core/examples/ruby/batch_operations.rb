import java.util.*;

import com.chain.api.*;
import com.chain.exception.*;
import com.chain.http.*;
import com.chain.signing.*;

class BatchOperations {
  public static void main(String[] args) throws Exception {
    Client client = new Client();

    MockHsm.Key key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));

    // snippet asset-builders
    List<Asset.Builder> assetBuilders = Arrays.asList(
      new Asset.Builder()
        .setAlias("gold")
        .addRootXpub(key.xpub)
        .setQuorum(1),
      new Asset.Builder()
        .setAlias("silver")
        .addRootXpub(key.xpub)
        .setQuorum(1),
      new Asset.Builder()
        .setAlias("bronze")
        .addRootXpub(key.xpub)
        .setQuorum(0)
    );
    // endsnippet

    // snippet asset-create-batch
    BatchResponse<Asset> assetBatch = Asset.createBatch(client, assetBuilders);
    // endsnippet

    // snippet asset-create-handle-errors
    for (int i = 0; i < assetBatch.size(); i++) {
      if (assetBatch.isError(i)) {
        APIException error = assetBatch.errorsByIndex().get(i);
        System.out.println("asset " + i + " error: " + error);
      } else {
        Asset asset = assetBatch.successesByIndex().get(i);
        System.out.println("asset " + i + " created, ID: " + asset.id);
      }
    }
    // endsnippet

    // snippet nondeterministic-errors
    assetBuilders = Arrays.asList(
      new Asset.Builder()
        .setAlias("platinum")
        .addRootXpub(key.xpub)
        .setQuorum(1),
      new Asset.Builder()
        .setAlias("platinum")
        .addRootXpub(key.xpub)
        .setQuorum(1),
      new Asset.Builder()
        .setAlias("platinum")
        .addRootXpub(key.xpub)
        .setQuorum(1)
    );
    // endsnippet

    assetBatch = Asset.createBatch(client, assetBuilders);

    for (int i = 0; i < assetBatch.size(); i++) {
      if (assetBatch.isError(i)) {
        APIException error = assetBatch.errorsByIndex().get(i);
        System.out.println("asset " + i + " error: " + error);
      } else {
        Asset asset = assetBatch.successesByIndex().get(i);
        System.out.println("asset " + i + " created, ID: " + asset.id);
      }
    }

    new Account.Builder()
      .setAlias("alice")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    new Account.Builder()
      .setAlias("bob")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

    // snippet batch-build-builders
    List<Transaction.Builder> txBuilders = Arrays.asList(
      new Transaction.Builder()
        .addAction(new Transaction.Action.Issue()
          .setAssetAlias("gold")
          .setAmount(100)
        ).addAction(new Transaction.Action.ControlWithAccount()
          .setAccountAlias("alice")
          .setAssetAlias("gold")
          .setAmount(100)
        ),
      new Transaction.Builder()
        .addAction(new Transaction.Action.Issue()
          .setAssetAlias("not-a-real-asset")
          .setAmount(100)
        ).addAction(new Transaction.Action.ControlWithAccount()
          .setAccountAlias("alice")
          .setAssetAlias("not-a-real-asset")
          .setAmount(100)
        ),
      new Transaction.Builder()
        .addAction(new Transaction.Action.Issue()
          .setAssetAlias("silver")
          .setAmount(100)
        ).addAction(new Transaction.Action.ControlWithAccount()
          .setAccountAlias("alice")
          .setAssetAlias("silver")
          .setAmount(100)
        )
    );
    // endsnippet

    // snippet batch-build-handle-errors
    BatchResponse<Transaction.Template> buildTxBatch = Transaction.buildBatch(client, txBuilders);

    for(Map.Entry<Integer, APIException> err : buildTxBatch.errorsByIndex().entrySet()) {
      System.out.println("Error building transaction " + err.getKey() + ": " + err.getValue());
    }
    // endsnippet

    // snippet batch-sign
    BatchResponse<Transaction.Template> signTxBatch = HsmSigner.signBatch(buildTxBatch.successes());

    for(Map.Entry<Integer, APIException> err : signTxBatch.errorsByIndex().entrySet()) {
      System.out.println("Error signing transaction " + err.getKey() + ": " + err.getValue());
    }
    // endsnippet

    // snippet batch-submit
    BatchResponse<Transaction.SubmitResponse> submitTxBatch = Transaction.submitBatch(client, signTxBatch.successes());

    for(Map.Entry<Integer, APIException> err : submitTxBatch.errorsByIndex().entrySet()) {
      System.out.println("Error submitting transaction " + err.getKey() + ": " + err.getValue());
    }

    for(Map.Entry<Integer, Transaction.SubmitResponse> success : submitTxBatch.successesByIndex().entrySet()) {
      System.out.println("Transaction " + success.getKey() + " submitted, ID: " + success.getValue().id);
    }
    // endsnippet
  }
}
