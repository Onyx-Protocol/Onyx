import java.io.FileWriter;
import java.io.PrintWriter;
import java.net.URL;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

import com.chain.api.*;
import com.chain.api.MockHsm.Key;
import com.chain.http.Client;
import com.chain.signing.HsmSigner;

public class IouSettlement {
  public static void main(String[] args) throws Exception {
    String coreURL = System.getenv("CHAIN_API_URL");
    String accessToken = System.getenv("CHAIN_API_TOKEN");
    System.out.println(coreURL);
    System.out.println(accessToken);
    Client client = new Client(new URL(coreURL), accessToken);
    client.setConnectTimeout(10, TimeUnit.MINUTES);
    client.setReadTimeout(10, TimeUnit.MINUTES);
    client.setWriteTimeout(10, TimeUnit.MINUTES);
    bench(client);
    System.exit(0);
  }

  static void bench(Client client) throws Exception {
    MockHsm.Key dealerAccountKey = MockHsm.Key.create(client);
    MockHsm.Key dealerIssuerKey = MockHsm.Key.create(client);
    MockHsm.Key northBankIssuerKey = MockHsm.Key.create(client);
    MockHsm.Key southBankIssuerKey = MockHsm.Key.create(client);
    loadKeys(client);

    new Asset.Builder()
        .setAlias("dealerusd")
        .addRootXpub(dealerIssuerKey.xpub)
        .setQuorum(1)
        .addTag("entity", "dealer")
        .addTag("currency", "usd")
        .create(client);

    new Asset.Builder()
        .setAlias("nbusd")
        .addRootXpub(northBankIssuerKey.xpub)
        .setQuorum(1)
        .addTag("currency", "usd")
        .create(client);

    new Asset.Builder()
        .setAlias("sbusd")
        .addRootXpub(southBankIssuerKey.xpub)
        .setQuorum(1)
        .addTag("currency", "usd")
        .create(client);

    new Account.Builder()
        .setAlias("dealer")
        .addRootXpub(dealerAccountKey.xpub)
        .setQuorum(1)
        .create(client);

    new Account.Builder()
        .setAlias("nb")
        .addRootXpub(dealerAccountKey.xpub)
        .setQuorum(1)
        .create(client);

    new Account.Builder()
        .setAlias("sb")
        .addRootXpub(dealerAccountKey.xpub)
        .setQuorum(1)
        .create(client);

    Dealer dealer = new Dealer(client, getAccount(client, "dealer"), getAsset(client, "dealerusd"));
    Bank northBank = new Bank(client, dealer, getAsset(client, "nbusd"), getAccount(client, "nb"));
    Bank southBank = new Bank(client, dealer, getAsset(client, "sbusd"), getAccount(client, "sb"));

    Corp acme = new Corp("acme", northBank, client);
    Corp zzzz = new Corp("zzzz", southBank, client);

    // Number of threads per corp.
    final int nthread = 20;

    // Create enough txs to equal (one day's worth of activity at 10 tx/s).
    //final int ntxTotal = 10 * 60*60*24; // should be multiple of 2*nthread
    final int ntxTotal = 10 * 60 * 60 * 6; // start with six hour's worth

    // # of txs to submit in one request
    final int batchSize = 10;

    // # of batches per corp
    final int nbatches = ntxTotal / batchSize / 2;

    Instant start = Instant.now();

    AtomicInteger processed = new AtomicInteger(0);
    ExecutorService pool = Executors.newFixedThreadPool(2 * nthread);
    List<Callable<Integer>> x = new ArrayList<>();
    for (int b = 0; b < nbatches; b++) {
      x.add(
          () -> {
            acme.pay(zzzz, batchSize);
            int v = processed.getAndAdd(batchSize) + batchSize;
            if (v % 1000 == 0) {
              long elapsed = Duration.between(start, Instant.now()).toMillis();
              double tps = (double) v / elapsed * 1000.0;
              System.out.printf(
                  "%d / %d (%d%%) %.2f tx/sec\n", v, ntxTotal, 100 * v / ntxTotal, tps);
            }
            return 1;
          });
    }
    for (int b = 0; b < nbatches; b++) {
      x.add(
          () -> {
            zzzz.pay(acme, batchSize);
            int v = processed.getAndAdd(batchSize) + batchSize;
            if (v % 1000 == 0) {
              long elapsed = Duration.between(start, Instant.now()).toMillis();
              double tps = (double) v / elapsed * 1000.0;
              System.out.printf(
                  "%d / %d (%d%%) %.2f tx/sec\n", v, ntxTotal, 100 * v / ntxTotal, tps);
            }
            return 1;
          });
    }

    Instant tstart = Instant.now();
    List<Future<Integer>> futures = pool.invokeAll(x);
    for (int b = 0; b < 2 * nbatches; b++) {
      futures.get(b).get();
    }
    Instant tend = Instant.now();
    System.out.println("done transacting.");
    long elapsed = Duration.between(tstart, tend).toMillis();
    System.out.printf("elapsed time %dms\n", elapsed);
    PrintWriter stats = new PrintWriter(new FileWriter("stats.json"));
    stats.printf("{\"elapsed_ms\": %d, \"txs\": %d}\n", elapsed, ntxTotal);
    stats.close();
  }

  static Asset getAsset(Client client, String id) throws Exception {
    String q = String.format("alias='%s'", id);
    Asset.Items assets = new Asset.QueryBuilder().setFilter(q).execute(client);
    if (assets.list.size() != 1) {
      throw new Exception(String.format("missing asset: %s", id));
    }
    return assets.list.get(0);
  }

  static Account getAccount(Client client, String id) throws Exception {
    Account.Items accounts =
        new Account.QueryBuilder().setFilter(String.format("alias='%s'", id)).execute(client);
    if (accounts.list.size() != 1) {
      throw new Exception(String.format("missing account: %s", id));
    }
    return accounts.list.get(0);
  }

  static void loadKeys(Client client) throws Exception {
    Key.Items keys = new MockHsm.Key.QueryBuilder().execute(client);
    while (keys.hasNext()) {
      Key k = keys.next();
      HsmSigner.addKey(k.xpub, MockHsm.getSignerClient(client));
    }
  }
}

class Bank {
  Asset asset;
  Account account;
  Dealer dealer;
  Client client;

  Bank(Client client, Dealer dealer, Asset asset, Account account) {
    this.client = client;
    this.dealer = dealer;
    this.asset = asset;
    this.account = account;
  }

  void pay(Corp corp, Corp payee, Integer times) throws Exception {
    List<Transaction.Builder> builders = new ArrayList<Transaction.Builder>();
    for (int i = 0; i < times; i++) {
      builders.add(
          new Transaction.Builder()
              .addAction(
                  new Transaction.Action.Issue()
                      .setAssetId(this.asset.id)
                      .setAmount(1)
                      .setReferenceData(corp.ref()))
              .addAction(new Transaction.Action.Issue().setAssetId(dealer.usd.id).setAmount(1))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountId(dealer.account.id)
                      .setAssetId(this.asset.id)
                      .setAmount(1))
              .addAction(
                  new Transaction.Action.ControlWithAccount()
                      .setAccountId(payee.bank.account.id)
                      .setAssetId(this.dealer.usd.id)
                      .setAmount(1)
                      .setReferenceData(payee.ref())));
    }
    List<Transaction.Template> templates = Transaction.buildBatch(client, builders).successes();
    List<Transaction.Template> signedTemplates = HsmSigner.signBatch(templates).successes();
    List<Transaction.SubmitResponse> txs =
        Transaction.submitBatch(client, signedTemplates, "confirmed").successes();
  }
}

class Corp {
  String name;
  Bank bank;
  Client client;

  Corp(String name, Bank bank, Client client) {
    this.name = name;
    this.bank = bank;
    this.client = client;
  }

  void pay(Corp payee, Integer times) throws Exception {
    this.bank.pay(this, payee, times);
  }

  HashMap<String, Object> ref() {
    HashMap<String, Object> m = new HashMap<String, Object>();
    m.put("corporate", this.name);
    return m;
  }
}

class Dealer {
  Account account;
  Asset usd;
  Client client;

  Dealer(Client client, Account account, Asset usd) {
    this.account = account;
    this.usd = usd;
    this.client = client;
  }
}
