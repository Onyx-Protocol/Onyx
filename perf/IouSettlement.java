import java.io.FileWriter;
import java.io.PrintWriter;
import java.math.BigInteger;
import java.net.URL;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
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
    Context ctx = new Context(new URL(System.getenv("CHAIN_API_URL")));
    ctx.setConnectTimeout(10, TimeUnit.MINUTES);
    ctx.setReadTimeout(10, TimeUnit.MINUTES);
    ctx.setWriteTimeout(10, TimeUnit.MINUTES);
    setup(ctx);
    bench(ctx);
    System.exit(0);
  }

  static void setup(Context ctx) throws Exception {
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

    new Account.Builder().setAlias("nb").addRootXpub(dealerAccountKey.xpub).setQuorum(1).create(ctx);

    new Account.Builder().setAlias("sb").addRootXpub(dealerAccountKey.xpub).setQuorum(1).create(ctx);

    Dealer dealer = new Dealer(ctx, getAccount(ctx, "dealer"), getAsset(ctx, "dealerusd"));
    Bank northBank = new Bank(ctx, dealer, getAsset(ctx, "nbusd"), getAccount(ctx, "nb"));
    Bank southBank = new Bank(ctx, dealer, getAsset(ctx, "sbusd"), getAccount(ctx, "sb"));

    Corp acme = new Corp("acme", northBank, ctx);
    Corp zzzz = new Corp("zzzz", southBank, ctx);

    // Number of threads per corp.
    final int nthread = 20;

    // Create enough txs to equal (one day's worth of activity at 10 tx/s).
    //final int ntxTotal = 10 * 60*60*24; // should be multiple of 2*nthread
    final int ntxTotal = 10 * 60 * 30; // start with a half hour's worth

    // Number of transactions per thread.
    // nthread * 2 corps * ntx = ntxTotal
    final int ntx = ntxTotal / 2 / nthread;

    ExecutorService pool = Executors.newFixedThreadPool(2 * nthread);
    List<Callable<Integer>> x = new ArrayList<>();
    for (int t = 0; t < nthread; t++) {
      x.add(
          () -> {
            for (int i = 0; i < ntx; i++) {
              if (i % 10 == 0) {
                System.out.printf("%d / %d (%d%%)\ntx", i, ntx, 100 * i / ntx);
              }
              Instant ti = Instant.now();
              acme.pay(zzzz, i + 1);
              Duration elapsed = Duration.between(ti, Instant.now());
              long sleep = 100 - elapsed.toMillis();
              if (sleep > 0) {
                Thread.sleep(sleep); // offer at most 10 calls/sec.
              }
            }
            return 1;
          });
    }
    for (int t = 0; t < nthread; t++) {
      x.add(
          () -> {
            for (int i = 0; i < ntx; i++) {
              if (i % 10 == 0) {
                System.out.printf("%d / %d (%d%%)\ntx", i, ntx, 100 * i / ntx);
              }
              Instant ti = Instant.now();
              zzzz.pay(acme, i + 1);
              Duration elapsed = Duration.between(ti, Instant.now());
              long sleep = 100 - elapsed.toMillis();
              if (sleep > 0) {
                Thread.sleep(sleep); // offer at most 10 calls/sec.
              }
            }
            return 1;
          });
    }
    List<Future<Integer>> futures = pool.invokeAll(x);
    for (int t = 0; t < 2 * nthread; t++) {
      futures.get(t).get();
    }
    System.out.println("setup done");
  }

  static void bench(Context ctx) throws Exception {
    loadKeys(ctx);

    Dealer dealer = new Dealer(ctx, getAccount(ctx, "dealer"), getAsset(ctx, "dealerusd"));
    Bank northBank = new Bank(ctx, dealer, getAsset(ctx, "nbusd"), getAccount(ctx, "nb"));
    Bank southBank = new Bank(ctx, dealer, getAsset(ctx, "sbusd"), getAccount(ctx, "sb"));

    Corp acme = new Corp("acme", northBank, ctx);
    Corp zzzz = new Corp("zzzz", southBank, ctx);

    // Target is 10 tx/s total.
    // We are going to do 5tx/s per corp.
    // We'll also do 5 threads per corp,
    // so each thread should do 1 tx/s.

    // Number of threads per corp.
    final int nthread = 5;

    // Min time (in ms) between attempts to send each transaction in a thread.
    final long txperiod = 1000;

    //final int ntxTotal = 10 * 60*60*24; // should be multiple of 2*nthread
    final int ntxTotal = 10 * 60 * 30; // start with a half hour's worth

    // Number of transactions per thread.
    // nthread * 2 corps * ntx = ntxTotal
    final int ntx = ntxTotal / 2 / nthread;

    ExecutorService pool = Executors.newFixedThreadPool(2 * nthread);
    List<Callable<Integer>> x = new ArrayList<>();
    for (int t = 0; t < nthread; t++) {
      x.add(
          () -> {
            for (int i = 0; i < ntx; i++) {
              if (i % 10 == 0) {
                System.out.printf("%d / %d (%d%%)\ntx", i, ntx, 100 * i / ntx);
              }
              Instant ti = Instant.now();
              acme.pay(zzzz, i + 1);
              Duration elapsed = Duration.between(ti, Instant.now());
              long sleep = txperiod - elapsed.toMillis();
              if (sleep > 0) {
                Thread.sleep(sleep); // offer at most 1/txperiod calls/sec.
              }
            }
            return 1;
          });
    }
    for (int t = 0; t < nthread; t++) {
      x.add(
          () -> {
            for (int i = 0; i < ntx; i++) {
              if (i % 10 == 0) {
                System.out.printf("%d / %d (%d%%)\ntx", i, ntx, 100 * i / ntx);
              }
              Instant ti = Instant.now();
              zzzz.pay(acme, i + 1);
              Duration elapsed = Duration.between(ti, Instant.now());
              long sleep = txperiod - elapsed.toMillis();
              if (sleep > 0) {
                Thread.sleep(sleep); // offer at most 1/txperiod calls/sec.
              }
            }
            return 1;
          });
    }
    Instant tstart = Instant.now();
    List<Future<Integer>> futures = pool.invokeAll(x);
    for (int t = 0; t < 2 * nthread; t++) {
      futures.get(t).get();
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

  void pay(Corp corp, Corp payee, Integer amount) throws Exception {
    Transaction.Template txTmpl =
        new Transaction.Builder()
            .issueById(this.asset.id, BigInteger.valueOf(amount), corp.ref())
            .issueById(dealer.usd.id, BigInteger.valueOf(amount), null)
            .controlWithAccountById(
                dealer.account.id, this.asset.id, BigInteger.valueOf(amount), null)
            .controlWithAccountById(
                payee.bank.account.id, this.dealer.usd.id, BigInteger.valueOf(amount), payee.ref())
            .build(ctx);

    List<Transaction.Template> signedTpls = HsmSigner.sign(Arrays.asList(txTmpl));
    List<Transaction.SubmitResponse> txs = Transaction.submit(ctx, signedTpls);
    for (Transaction.SubmitResponse sr : txs) {
      System.out.println(String.format("Created tx id=%s", sr.id));
    }
  }

  void incoming() throws Exception {
    Transaction.Items transactions =
        new Transaction.QueryBuilder()
            .setFilter("outputs(account_id=$1)")
            .addFilterParameter(this.account.id)
            .execute(ctx);
    while (transactions.hasNext()) {
      Transaction tx = transactions.next();
    }
  }

  void outgoing() throws Exception {
    Transaction.Items transactions =
        new Transaction.QueryBuilder()
            .setFilter("inputs(action='issuance' AND asset_id = $1)")
            .addFilterParameter(this.account.id)
            .execute(ctx);
    while (transactions.hasNext()) {
      Transaction tx = transactions.next();
    }
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

  void pay(Corp payee, Integer amount) throws Exception {
    this.bank.pay(this, payee, amount);
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

  void reportAllPayments() throws Exception {
    Transaction.Items transactions =
        new Transaction.QueryBuilder().setFilter("inputs(action='issue')").execute(this.ctx);

    System.out.println("report: all dealer payments");
    while (transactions.hasNext()) {
      Transaction tx = transactions.next();
      System.out.printf("\ttx: %s\n", tx.id);
    }
  }

  void reportSettlements() throws Exception {
    Transaction.Items transactions =
        new Transaction.QueryBuilder().setFilter("outputs(action='retire')").execute(this.ctx);
  }

  void reportCurrencyExposure() throws Exception {
    Balance.Items balanceItems;
    HashMap<String, BigInteger> exposure = new HashMap<String, BigInteger>();

    //Incoming
    balanceItems =
        new Balance.QueryBuilder()
            .setFilter("account_id='" + this.account.id + "' AND asset_tags.currency=$1")
            .setTimestamp(System.currentTimeMillis())
            .execute(this.ctx);

    while (balanceItems.hasNext()) {
      Balance balance = balanceItems.next();
      String currency = balance.sumBy.get("asset_tags.currency");
      BigInteger x = BigInteger.valueOf(0);
      if (exposure.containsKey(currency)) {
        x = exposure.get(currency);
      }
      exposure.put(currency, x.add(balance.amount));
    }

    //Outgoing
    balanceItems =
        new Balance.QueryBuilder()
            .setFilter("asset_tags.entity='dealer' AND asset_tags.currency=$1")
            .setTimestamp(System.currentTimeMillis())
            .execute(this.ctx);
    while (balanceItems.hasNext()) {
      Balance balance = balanceItems.next();
      String currency = balance.sumBy.get("asset_tags.currency");
      exposure.put(currency, exposure.get(currency).subtract(balance.amount));
    }

    System.out.println("report: dealer currency exposure");
    for (String currency : exposure.keySet()) {
      System.out.printf("\tcurrency: %s amount: %d\n", currency, exposure.get(currency));
    }
  }
}
