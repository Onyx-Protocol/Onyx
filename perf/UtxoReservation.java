import java.io.FileWriter;
import java.io.PrintWriter;
import java.net.URL;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Random;
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

public class UtxoReservation {
  public static void main(String[] args) throws Exception {
    String coreURL = System.getenv("CHAIN_API_URL");
    String accessToken = System.getenv("CHAIN_API_TOKEN");
    System.out.println(coreURL);
    System.out.println(accessToken);
    Client client = new Client(new URL(coreURL), accessToken);
    client.setConnectTimeout(10, TimeUnit.MINUTES);
    client.setReadTimeout(10, TimeUnit.MINUTES);
    client.setWriteTimeout(10, TimeUnit.MINUTES);
    setup(client);
    transact(client);
    System.exit(0);
  }

  static final int utxosPerDenomination = 100;
  static final int utxoDenominations = 5;

  static long totalPerAccount() {
    long total = 0;
    for (int i = 0; i < utxoDenominations; i++) {
      total += utxosPerDenomination * Math.pow(10, i);
    }
    return total;
  }

  // setup issues an asset into a couple of accounts at
  // at several denominations.
  static void setup(Client client) throws Exception {
    MockHsm.Key centralBankIssuerKey = MockHsm.Key.create(client);
    MockHsm.Key aliceAccountKey = MockHsm.Key.create(client);
    MockHsm.Key bobAccountKey = MockHsm.Key.create(client);
    loadKeys(client);

    Asset currency =
        new Asset.Builder()
            .setAlias("currency")
            .addRootXpub(centralBankIssuerKey.xpub)
            .setQuorum(1)
            .create(client);

    Account alice =
        new Account.Builder()
            .setAlias("alice")
            .addRootXpub(aliceAccountKey.xpub)
            .setQuorum(1)
            .create(client);

    Account bob =
        new Account.Builder()
            .setAlias("bob")
            .addRootXpub(bobAccountKey.xpub)
            .setQuorum(1)
            .create(client);

    // Issue some currency to Alice & Bob at several amounts per utxos.
    Transaction.Builder builder =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.Issue()
                    .setAssetId(currency.id)
                    .setAmount(2 * totalPerAccount()));

    for (int i = 0; i < utxoDenominations; i++) {
      long denominationAmount = (long) Math.pow(10, i);
      for (int j = 0; j < utxosPerDenomination; j++) {
        builder
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAssetId(currency.id)
                    .setAmount(denominationAmount)
                    .setAccountId(alice.id))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAssetId(currency.id)
                    .setAmount(denominationAmount)
                    .setAccountId(bob.id));
      }
    }
    Transaction.Template template = builder.build(client);
    Transaction.Template signedTemplate = HsmSigner.sign(template);
    Transaction.SubmitResponse tx = Transaction.submit(client, signedTemplate, "confirmed");
  }

  static void transact(Client client) throws Exception {
    loadKeys(client);
    Asset currency = getAsset(client, "currency");
    Account alice = getAccount(client, "alice");
    Account bob = getAccount(client, "bob");

    final int iterations = 300; // 5 minutes
    final int concurrentPayments = 250;
    final int maxPerPayment = (int) totalPerAccount() / concurrentPayments;

    Random r = new Random();
    ExecutorService pool = Executors.newFixedThreadPool(2 * concurrentPayments);

    List<Callable<Integer>> x = new ArrayList<>();
    for (int i = 0; i < iterations; i++) {
      for (int j = 0; j < concurrentPayments; j++) {
        long amount = (long) r.nextInt(maxPerPayment - 1) + 1;
        x.add(
            () -> {
              pay(client, alice, bob, currency, amount);
              return 1;
            });
        x.add(
            () -> {
              pay(client, bob, alice, currency, amount);
              return 1;
            });
      }
    }

    Instant tstart = Instant.now();
    List<Future<Integer>> futures = pool.invokeAll(x);
    for (int i = 0; i < futures.size(); i++) {
      futures.get(i).get();
    }
    Instant tend = Instant.now();
    System.out.println("done transacting.");
    long elapsed = Duration.between(tstart, tend).toMillis();
    System.out.printf("elapsed time %dms\n", elapsed);
    PrintWriter stats = new PrintWriter(new FileWriter("stats.json"));
    stats.printf("{\"elapsed_ms\": %d, \"txs\": %d}\n", elapsed, futures.size());
    stats.close();
  }

  static void pay(Client client, Account from, Account to, Asset asset, long amount)
      throws Exception {
    Transaction.Builder builder =
        new Transaction.Builder()
            .addAction(
                new Transaction.Action.SpendFromAccount()
                    .setAssetId(asset.id)
                    .setAmount(amount)
                    .setAccountId(from.id))
            .addAction(
                new Transaction.Action.ControlWithAccount()
                    .setAssetId(asset.id)
                    .setAmount(amount)
                    .setAccountId(to.id));
    Transaction.Template template = builder.build(client);
    Transaction.Template signedTemplate = HsmSigner.sign(template);
    Transaction.SubmitResponse tx = Transaction.submit(client, signedTemplate, "confirmed");
  }

  static Asset getAsset(Client client, String alias) throws Exception {
    Asset.Items assets =
        new Asset.QueryBuilder().setFilter("alias = $1").addFilterParameter(alias).execute(client);

    if (assets.list.size() != 1) {
      throw new Exception(String.format("missing asset: %s", alias));
    }
    return assets.list.get(0);
  }

  static Account getAccount(Client client, String alias) throws Exception {
    Account.Items accounts =
        new Account.QueryBuilder()
            .setFilter("alias = $1")
            .addFilterParameter(alias)
            .execute(client);
    if (accounts.list.size() != 1) {
      throw new Exception(String.format("missing account: %s", alias));
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
