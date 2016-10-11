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
import com.chain.http.Context;
import com.chain.signing.HsmSigner;

public class IouSettlement {
  public static void main(String[] args) throws Exception {
    String coreURL = System.getenv("CHAIN_API_URL");
    String accessToken = System.getenv("CHAIN_API_TOKEN");
    System.out.println(coreURL);
    System.out.println(accessToken);
    Context ctx = new Context(new URL(coreURL), accessToken);
    ctx.setConnectTimeout(10, TimeUnit.MINUTES);
    ctx.setReadTimeout(10, TimeUnit.MINUTES);
    ctx.setWriteTimeout(10, TimeUnit.MINUTES);
    bench(ctx);
    System.exit(0);
  }

  static void bench(Context ctx) throws Exception {
    MockHsm.Key dealerAccountKey = MockHsm.Key.create(ctx);
    MockHsm.Key dealerIssuerKey = MockHsm.Key.create(ctx);
    MockHsm.Key northBankIssuerKey = MockHsm.Key.create(ctx);
    MockHsm.Key southBankIssuerKey = MockHsm.Key.create(ctx);
    loadKeys(ctx);

    new Asset.Builder()
        .setAlias("dealerusd")
        .addRootXpub(dealerIssuerKey.xpub)
        .setQuorum(1)
        .addTag("entity", "dealer")
        .addTag("currency", "usd")
        .create(ctx);

    new Asset.Builder()
        .setAlias("nbusd")
        .addRootXpub(northBankIssuerKey.xpub)
        .setQuorum(1)
        .addTag("currency", "usd")
        .create(ctx);

    new Asset.Builder()
        .setAlias("sbusd")
        .addRootXpub(southBankIssuerKey.xpub)
        .setQuorum(1)
        .addTag("currency", "usd")
        .create(ctx);

    new Account.Builder()
        .setAlias("dealer")
        .addRootXpub(dealerAccountKey.xpub)
        .setQuorum(1)
        .create(ctx);

    new Account.Builder()
        .setAlias("nb")
        .addRootXpub(dealerAccountKey.xpub)
        .setQuorum(1)
        .create(ctx);

    new Account.Builder()
        .setAlias("sb")
        .addRootXpub(dealerAccountKey.xpub)
        .setQuorum(1)
        .create(ctx);

    Dealer dealer = new Dealer(ctx, getAccount(ctx, "dealer"), getAsset(ctx, "dealerusd"));
    Bank northBank = new Bank(ctx, dealer, getAsset(ctx, "nbusd"), getAccount(ctx, "nb"));
    Bank southBank = new Bank(ctx, dealer, getAsset(ctx, "sbusd"), getAccount(ctx, "sb"));

    Corp acme = new Corp("acme", northBank, ctx);
    Corp zzzz = new Corp("zzzz", southBank, ctx);

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
      x.add(() -> {
        acme.pay(zzzz, batchSize);
        int v = processed.getAndAdd(batchSize) + batchSize;
        if (v % 1000 == 0) {
          long elapsed = Duration.between(start, Instant.now()).toMillis();
          double tps = (double) v / elapsed * 1000.0;
          System.out.printf("%d / %d (%d%%) %.2f tx/sec\n", v, ntxTotal, 100 * v / ntxTotal, tps);
        }
        return 1;
      });
    }
    for (int b = 0; b < nbatches; b++) {
      x.add(() -> {
        zzzz.pay(acme, batchSize);
        int v = processed.getAndAdd(batchSize) + batchSize;
        if (v % 1000 == 0) {
          long elapsed = Duration.between(start, Instant.now()).toMillis();
          double tps = (double) v / elapsed * 1000.0;
          System.out.printf("%d / %d (%d%%) %.2f tx/sec\n", v, ntxTotal, 100 * v / ntxTotal, tps);
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

  static Asset getAsset(Context ctx, String id) throws Exception {
    String q = String.format("alias='%s'", id);
    Asset.Items assets = new Asset.QueryBuilder().setFilter(q).execute(ctx);
    if (assets.list.size() != 1) {
      throw new Exception(String.format("missing asset: %s", id));
    }
    return assets.list.get(0);
  }

  static Account getAccount(Context ctx, String id) throws Exception {
    Account.Items accounts =
        new Account.QueryBuilder().setFilter(String.format("alias='%s'", id)).execute(ctx);
    if (accounts.list.size() != 1) {
      throw new Exception(String.format("missing account: %s", id));
    }
    return accounts.list.get(0);
  }

  static void loadKeys(Context ctx) throws Exception {
    Key.Items keys = MockHsm.Key.list(ctx);
    while (keys.hasNext()) {
      Key k = keys.next();
      HsmSigner.addKey(k.xpub, k.hsmUrl);
    }
  }
}

class Bank {
  Asset asset;
  Account account;
  Dealer dealer;
  Context ctx;

  Bank(Context ctx, Dealer dealer, Asset asset, Account account) {
    this.ctx = ctx;
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
    List<Transaction.Template> templates = Transaction.buildBatch(ctx, builders);
    List<Transaction.Template> signedTemplates = HsmSigner.signBatch(templates);
    List<Transaction.SubmitResponse> txs = Transaction.submitBatch(ctx, signedTemplates);
  }
}

class Corp {
  String name;
  Bank bank;
  Context ctx;

  Corp(String name, Bank bank, Context ctx) {
    this.name = name;
    this.bank = bank;
    this.ctx = ctx;
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
  Context ctx;

  Dealer(Context ctx, Account account, Asset usd) {
    this.account = account;
    this.usd = usd;
    this.ctx = ctx;
  }
}
