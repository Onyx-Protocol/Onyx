import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.exception.APIException;
import com.chain.exception.BuildException;
import com.chain.exception.ChainException;
import com.chain.http.BatchResponse;
import com.chain.http.Client;
import com.chain.signing.HsmSigner;

import java.io.File;
import java.io.FileWriter;
import java.io.PrintWriter;
import java.net.URL;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicLong;

class TradeMirror {
    private TradeQueue queue;

    private Client client;
    private Asset dollars;
    private MockHsm.Key rootAssetKey;
    private MockHsm.Key rootAccountKey;

    // createdSecurities and createdAccounts cache the set of securities and
    // accounts that we've already created on the Chain Core. For this demo,
    // securities and accounts are lazily created on Chain Core as we encounter
    // them. In a real system, these would already exist by the time we're
    // processing trades.
    private ConcurrentHashMap<String,Boolean> createdSecurities;
    private ConcurrentHashMap<String,Boolean> createdAccounts;
    private ConcurrentHashMap<String,AtomicLong> balances;

    public static void main(String[] args) throws Exception {
        String coreURL = System.getenv("CHAIN_API_URL");
        String accessToken = System.getenv("CHAIN_API_TOKEN");

        System.out.println(coreURL);
        System.out.println(accessToken);

        Client client = new Client(new URL(coreURL), accessToken);
        client.setConnectTimeout(10, TimeUnit.MINUTES);
        client.setReadTimeout(10, TimeUnit.MINUTES);
        client.setWriteTimeout(10, TimeUnit.MINUTES);

        final String mockQueueFilename = "trades1000k.csv";
        TradeQueue queue = MockTradeQueue.fromCSV(mockQueueFilename);

        TradeMirror mirror = new TradeMirror(queue, client);
        int count = mirror.run();
        System.out.printf("simulated %d trades\n", count);
    }

    public TradeMirror(TradeQueue q, Client c) {
        this.queue = q;
        this.client = c;
        this.createdSecurities = new ConcurrentHashMap<String,Boolean>();
        this.createdAccounts = new ConcurrentHashMap<String,Boolean>();
        this.balances = new ConcurrentHashMap<String,AtomicLong>();
    }

    public int run() throws Exception {
        rootAssetKey = MockHsm.Key.create(client, "root_asset");
        rootAccountKey = MockHsm.Key.create(client, "root_account");
        // Load the above keys, plus any other keys we have from assets or
        // accounts that were created before this run.
        this.loadKeys(client);

        // Create a new asset to represent USD.
        try {
            Map<String, Object> usdDefinition = new HashMap<String, Object>();
            usdDefinition.put("currency", "USD");
            dollars = new Asset.Builder()
                    .setDefinition(usdDefinition)
                    .setAlias("usd")
                    .addRootXpub(rootAssetKey.xpub)
                    .setQuorum(1)
                    .create(client);
        } catch (APIException ex) {
            if (!"CH050".equals(ex.code)) {
                throw ex;
            }
            // Error code CH050 indicates that an asset with the alias 'usd' already
            // exists. Retrieve it, instead of creating a new asset.
            Asset.Items items = new Asset.QueryBuilder()
                    .setFilter("alias = $1")
                    .setFilterParameters(Arrays.asList("usd"))
                    .execute(client);
            if (!items.hasNext()) {
                throw new Exception("couldn't find or create usd asset");
            }
            dollars = items.next();
        }

        // Simulate every trade on Chain Core.
        final int maxParallelTxs = 300;

        ExecutorService pool = Executors.newFixedThreadPool(maxParallelTxs);
        List<Callable<String>> x = new ArrayList<>();
        int count = 0;
        while (queue.hasNext()) {
            Trade trade = queue.next();
            x.add(() -> { return simulateTrade(trade); });
            count++;
        }

        Instant tstart = Instant.now();
        List<Future<String>> futures = pool.invokeAll(x);
        for (int i = 0; i < count; i++) {
            futures.get(i).get();
        }
        Instant tend = Instant.now();

        pool.shutdown();

        System.out.println("done transacting.");
        long elapsed = Duration.between(tstart, tend).toMillis();
        System.out.printf("elapsed time %dms\n", elapsed);
        PrintWriter stats = new PrintWriter(new FileWriter("stats.json"));
        stats.printf("{\"elapsed_ms\": %d, \"txs\": %d}\n", elapsed, count);
        stats.close();

        return count;
    }

    // simulateTrade replicates the given trade on Chain Core, creating any assets,
    // accounts and issuances necessary to make the trade happen. It returns the
    // transaction ID of the transaction recording the trade on the blockchain.
    private String simulateTrade(Trade trade) throws ChainException {
        // Automatically create Chain Core accounts for the buyer and seller if
        // they don't already have accounts.
        createAccounts(Arrays.asList(trade.buyerId, trade.sellerId));

        // Automatically create the security asset on Chain Core if it doesn't
        // already exist.
        createSecurity(trade.stockId);

        Transaction.Action receiveUSD = new Transaction.Action.ControlWithAccount()
                .setAccountAlias(trade.sellerId)
                .setAssetId(this.dollars.id)
                .setAmount(trade.totalPrice);
        Transaction.Action receiveSecurity = new Transaction.Action.ControlWithAccount()
                .setAccountAlias(trade.buyerId)
                .setAssetAlias(trade.stockId)
                .setAmount(trade.shares);

        Transaction.Action usdSource = null;
        Transaction.Action securitySource = null;

        if (canSpendFromBalance(trade.buyerId, "usd", trade.totalPrice)) {
            usdSource = new Transaction.Action.SpendFromAccount()
                    .setAccountAlias(trade.buyerId)
                    .setAssetId(this.dollars.id)
                    .setAmount(trade.totalPrice);
        } else {
            usdSource = new Transaction.Action.Issue()
                    .setAssetId(this.dollars.id)
                    .setAmount(trade.totalPrice);
        }

        if (canSpendFromBalance(trade.sellerId, trade.stockId, trade.shares)) {
            securitySource = new Transaction.Action.SpendFromAccount()
                .setAccountAlias(trade.sellerId)
                .setAssetAlias(trade.stockId)
                .setAmount(trade.shares);
        } else {
            securitySource = new Transaction.Action.Issue()
                .setAssetAlias(trade.stockId)
                .setAmount(trade.shares);
        }

        Transaction.Template template = new Transaction.Builder()
                .addAction(receiveUSD)
                .addAction(receiveSecurity)
                .addAction(usdSource)
                .addAction(securitySource)
                .build(client);
        Transaction.SubmitResponse resp = Transaction.submit(client, HsmSigner.sign(template), "confirmed");

        addToBalance(trade.sellerId, "usd", trade.totalPrice);
        addToBalance(trade.buyerId, trade.stockId, trade.shares);
        return resp.id;
    }

    private boolean canSpendFromBalance(String accountID, String assetID, long amount) {
        String balanceKey = accountID + "-" + assetID;
        AtomicLong balance = balances.get(balanceKey);
        if (balance == null) {
            return false;
        }

        // Atomically take amount from the balance and see if there's still
        // remaining balance.
        long newBalance = balance.addAndGet(-1 * amount);
        if (newBalance < 0) {
            // We overdrew. Put it back and return false.
            balance.addAndGet(amount);
            return false;
        }
        return true;
    }

    private void addToBalance(String accountAlias, String assetAlias, long amount) {
        String balanceKey = accountAlias + "-" + assetAlias;
        if (!balances.containsKey(balanceKey)) {
            // Initialize the balance for this account/asset combination.
            balances.putIfAbsent(balanceKey, new AtomicLong(0));
        }

        balances.get(balanceKey).addAndGet(amount);
    }

    private void createAccounts(Collection<String> exchangeAccountIds) throws ChainException {
        List<Account.Builder> builders = new ArrayList<>();
        for (String exchangeAccountId : exchangeAccountIds) {
            if (createdAccounts.containsKey(exchangeAccountId)) {
                continue;
            }
            builders.add(new Account.Builder()
                    .setQuorum(1)
                    .addRootXpub(rootAccountKey.xpub)
                    .setAlias(exchangeAccountId));
        }
        BatchResponse<Account> resp = Account.createBatch(client, builders);
        for (APIException ex : resp.errors()) {
            // If the error is CH050, we've already created an account in the
            // Chain Core for this account on the exchange. We can swallow the
            // exception. Otherwise, throw the exception.
            if (!"CH050".equals(ex.code)) {
                throw ex;
            }
        }
        for (String exchangeAccountId : exchangeAccountIds) {
            createdAccounts.put(exchangeAccountId, true);
        }
    }

    private void createSecurity(String exchangeSecurityId) throws ChainException {
        if (createdSecurities.containsKey(exchangeSecurityId)) {
            return;
        }

        Map<String,Object> definition = new HashMap<>();
        definition.put("exchange", "Alice's Security Exchange");
        definition.put("security_id", exchangeSecurityId);

        try {
            new Asset.Builder()
                    .setQuorum(1)
                    .addRootXpub(rootAssetKey.xpub)
                    .setAlias(exchangeSecurityId)
                    .setDefinition(definition)
                    .create(client);
        } catch (APIException ex) {
            // If the error is CH050, we've already created an asset for this
            // security and we can swallow the exception. Otherwise, re-throw
            // the exception.
            if (!"CH050".equals(ex.code)) {
                throw ex;
            }
        }
        createdSecurities.put(exchangeSecurityId, true);
    }

    private void loadKeys(Client client) throws Exception {
        MockHsm.Key.Items keys = new MockHsm.Key.QueryBuilder().execute(client);
        while (keys.hasNext()) {
            MockHsm.Key k = keys.next();
            HsmSigner.addKey(k.xpub, MockHsm.getSignerClient(client));
        }
    }

    interface TradeQueue {
        boolean hasNext();

        Trade next() throws Exception;
    }

    public static class Trade {
        long Id;
        long totalPrice;
        long shares;
        String buyerId;
        String sellerId;
        String stockId;
    }

    public static class MockTradeQueue implements TradeQueue {
        private java.util.Queue<Trade> trades;

        public static MockTradeQueue fromCSV(String filename) throws Exception {
            Scanner s = new Scanner(new File(filename));
            Queue<Trade> trades = new LinkedList<>();
            while (s.hasNextLine()) {
                String line = s.nextLine();
                String[] record = line.trim().split(",");
                if (record.length != 5) {
                    throw new Exception(String.format("error reading line: %s\n", line));
                }

                Trade t = new Trade();
                t.Id = trades.size();
                t.totalPrice = Long.parseLong(record[0]);
                t.shares = Long.parseLong(record[1]);
                t.buyerId = record[2];
                t.sellerId = record[3];
                t.stockId = record[4];
                trades.add(t);
            }

            MockTradeQueue q = new MockTradeQueue();
            q.trades = trades;
            System.out.printf("Loaded %d trades from file\n", q.trades.size());
            return q;
        }

        public boolean hasNext() {
            return this.trades.size() > 0;
        }

        public Trade next() throws Exception {
            return this.trades.poll();
        }
    }
}
