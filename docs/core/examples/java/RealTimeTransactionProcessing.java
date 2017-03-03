import com.chain.api.*;
import com.chain.http.*;
import com.chain.signing.*;

class RealTimeTransactionProcessing {
  public static void main(String[] args) throws Exception {
    final Client client = new Client();
    setup(client);

    // snippet processing-thread
    new Thread(new Runnable() {
      public void run() {
        processingLoop(client);
      }
    }).start();
    // endsnippet

    // snippet issue
    Transaction.Template issuance = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(100)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(100)
      ).build(client);

    Transaction.submit(client, HsmSigner.sign(issuance));
    // endsnippet

    Thread.sleep(1000);

    // snippet transfer
    Transaction.Template transfer = new Transaction.Builder()
      .addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(50)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(50)
      ).build(client);

    Transaction.submit(client, HsmSigner.sign(transfer));
    // endsnippet

    Thread.sleep(1000);
    System.exit(0);
  }

  public static void setup(Client client) throws Exception {
    MockHsm.Key key = MockHsm.Key.create(client);
    HsmSigner.addKey(key, MockHsm.getSignerClient(client));

    new Asset.Builder()
      .setAlias("gold")
      .addRootXpub(key.xpub)
      .setQuorum(1)
      .create(client);

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

    // snippet create-feed
    Transaction.Feed feed = Transaction.Feed.create(
      client,
      "local-transactions",
      "is_local='yes'"
    );
    // endsnippet
  }

  public static void processingLoop(Client client) {
    try {
      // snippet get-feed
      Transaction.Feed feed = Transaction.Feed.getByAlias(
        client,
        "local-transactions"
      );
      // endsnippet

      // snippet processing-loop
      while (true) {
        Transaction tx = feed.next(client);
        processTransaction(tx);
        feed.ack(client);
      }
      // endsnippet
    } catch (Exception e) {
      throw new RuntimeException(e);
    }
  }

  // snippet processor-method
  public static void processTransaction(Transaction tx) {
    System.out.println("New transaction at " + tx.timestamp);
    System.out.println("\tID: " + tx.id);

    for (int i = 0; i < tx.inputs.size(); i++) {
      Transaction.Input input = tx.inputs.get(i);
      System.out.println("\tInput " + i);
      System.out.println("\t\tType: " + input.type);
      System.out.println("\t\tAsset: " + input.assetAlias);
      System.out.println("\t\tAmount: " + input.amount);
      System.out.println("\t\tAccount: " + input.accountAlias);
    }

    for (int i = 0; i < tx.outputs.size(); i++) {
      Transaction.Output output = tx.outputs.get(i);
      System.out.println("\tOutput " + i);
      System.out.println("\t\tType: " + output.type);
      System.out.println("\t\tPurpose: " + output.purpose);
      System.out.println("\t\tAsset: " + output.assetAlias);
      System.out.println("\t\tAmount: " + output.amount);
      System.out.println("\t\tAccount: " + output.accountAlias);
    }
  }
  // endsnippet
}
