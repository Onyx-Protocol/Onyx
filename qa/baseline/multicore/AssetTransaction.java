package chain.qa.baseline.multicore;

import java.net.URL;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.concurrent.Callable;
import java.util.concurrent.Executors;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.chain.*;

import chain.qa.*;

/**
 * AssetTransaction tests asset transactions on a multi-core network.
 */
public class AssetTransaction {
	// first core
	private static TestClient c;
	private static String issuerID;
	private static String managerID;
	private static String assetID;
	// second core
	private static TestClient sc;
	private static String secondIssuerID;
	private static String secondManagerID;
	private static String secondAssetID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID, TestClient secondClient, String spID)
	throws Exception {
		// setup first core
		c = client;
		issuerID = TestUtils.createIssuer(c, pID, "Transaction");
		managerID = TestUtils.createManager(c, pID, "Transaction");
		assetID = TestUtils.createAsset(c, issuerID, "Transaction");

		// setup second core
		sc = secondClient;
		secondIssuerID = TestUtils.createIssuer(sc, spID, "Transaction Second");
		secondManagerID = TestUtils.createManager(sc, spID, "Transaction Second");
		secondAssetID = TestUtils.createAsset(sc, secondIssuerID, "Transaction");

		// assertions
		assert testOneWayCrossCore();
		assert testAtomicSwapCrossCore();
		assert testForeignAssetTransaction();
		assert testDoubleSpend();
	}

	/**
	 * Executes a one way payment between accounts on separate cores.
	 * Asset is sent from its originating core. The assertions
	 * check that each core's manager accounts for the transaction.
	 */
	public static boolean testOneWayCrossCore()
	throws Exception {
		// create first core's account
		String acctID = TestUtils.createAccount(c, managerID, "One Way");

		// create second core's account
		String secondAcctID = TestUtils.createAccount(sc, secondManagerID, "One Way");
		String addr = TestUtils.createAddress(sc, secondAcctID);

		// issue 1000 units of asset to first core's account
		String issueID = TestUtils.issue(c, assetID, acctID, 1000);

		// transfer 600 units of asset to second core's account
		String txID = TestUtils.transact(c, assetID, acctID, addr, 600);
		TestUtils.waitForPropagation(sc, txID);
		System.out.printf("Transacted cross core. ID=%s\n", txID);


		// validate first core's account balance
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 400);
		TestUtils.validateAccountBalance(c, acctID, balances);

		// validate second core's account balance
		balances = new HashMap<String, Integer>();
		balances.put(assetID, 600);
		TestUtils.validateAccountBalance(sc, secondAcctID, balances);
		return true;
	}

	/**
	 * Executes an atomic swap between accounts on separate cores.
	 * Each asset is sent from accounts on its originating core. The assertions
	 * check that each core's manager accounts for the transaction.
	 */
	public static boolean testAtomicSwapCrossCore()
	throws Exception {
		// create first core's account
		String acctID = TestUtils.createAccount(c, managerID, "Atomic Swap");
		String addr = TestUtils.createAddress(c, acctID);

		// create second core's account
		String secondAcctID = TestUtils.createAccount(sc, secondManagerID, "Atomic Swap");
		String secondAddr = TestUtils.createAddress(sc, secondAcctID);

		// issue 1000 units of asset
		String issueID = TestUtils.issue(c, assetID, acctID, 1000);

		// issue 1000 units of secondAsset
		issueID = TestUtils.issue(sc, secondAssetID, secondAcctID, 1000);

		// build first part of transaction
		// send 750 units of asset to second core's account
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, acctID, BigInteger.valueOf(750));
		build.addAddressOutput(assetID, secondAddr, BigInteger.valueOf(750));
		Transactor.Transaction partialTx = c.buildTransaction(build);

		// build second part of transaction
		// send 250 units of secondAsset to first core's account
		List<Transactor.BuildRequest.Input> inputs = new ArrayList<Transactor.BuildRequest.Input>();
		List<Transactor.BuildRequest.Output> outputs = new ArrayList<Transactor.BuildRequest.Output>();
		build = new Transactor.BuildRequest(partialTx, inputs, outputs);
		build.addInput(secondAssetID, secondAcctID, BigInteger.valueOf(250));
		build.addAddressOutput(secondAssetID, addr, BigInteger.valueOf(250));
		Transactor.Transaction fullTx = sc.buildTransaction(build);

		// both clients must sign before submission
		c.signTransaction(fullTx);
		sc.signTransaction(fullTx);
		Transactor.SubmitResponse resp = sc.submitTransaction(fullTx);
		System.out.printf("Executed a cross core, atomic swap. ID=%s\n", resp.transactionID);


		// validate first core's account balances
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 250);
		balances.put(secondAssetID, 250);
		TestUtils.validateAccountBalance(c, acctID, balances);

		// validate second core's account balances
		balances = new HashMap<String, Integer>();
		balances.put(assetID, 750);
		balances.put(secondAssetID, 750);
		TestUtils.validateAccountBalance(sc, secondAcctID, balances);
		return true;
	}

	/**
	 * Executes a transfer of an asset created on the first core, between
	 * accounts on the second core. The assertions check that the second manager
	 * accounts for the transaction.
	 */
	public static boolean testForeignAssetTransaction()
	throws Exception {
		// create sender account
		String sndrID = TestUtils.createAccount(sc, secondManagerID, "Foreign Asset");
		String sAddr = TestUtils.createAddress(sc, sndrID);

		// create receiver account
		String rcvrID = TestUtils.createAccount(sc, secondManagerID, "Foreign Asset");
		String rAddr = TestUtils.createAddress(sc, rcvrID);

		// issue 1000 units of asset to sender
		String issueID = TestUtils.issueToAddress(c, assetID, sAddr, 1000);
		TestUtils.waitForPropagation(sc, issueID);

		// transfer 600 units of asset to receiver
		String txID = TestUtils.transact(sc, assetID, sndrID, rAddr, 600);
		System.out.printf("Transacted a foreign asset. ID=%s\n", txID);


		// validate sender's balance
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 400);
		TestUtils.validateAccountBalance(sc, sndrID, balances);

		// validate receiver's balance
		balances = new HashMap<String, Integer>();
		balances.put(assetID, 600);
		TestUtils.validateAccountBalance(sc, rcvrID, balances);
		return true;
	}

	/**
	 * Attempts to double spend a utxo on separate cores. The assertions check
	 * that only one of the transactions is committed to the blockchain.
	 */
	public static boolean testDoubleSpend()
	throws Exception {
		// create sender account
		String sndrID = TestUtils.createAccount(c, managerID, "Double Spend Sender");

		// create receiver account
		String rcvrID = TestUtils.createAccount(c, managerID, "Double Spend Receiver");
		String addr = TestUtils.createAddress(c, rcvrID);

		// create secondReceiver B account
		String secondRcvrID = TestUtils.createAccount(sc, secondManagerID, "Double Spend Receiver");
		String secondAddr = TestUtils.createAddress(sc, secondRcvrID);

		// issue 1000 units of asset to sender
		String issueID = TestUtils.issue(c, assetID, sndrID, 1000);
		System.out.println("Attempting a double spend:");

		// create a thread pool
		ExecutorService pool = Executors.newFixedThreadPool(2);

		// execute both transactions as Callable tasks within the thread pool
		List<Callable<Integer>> txs = Arrays.asList(
			() -> {
				String txID = TestUtils.transact(c, assetID, sndrID, addr, 1000);
				System.out.printf("\tID=%s\n", txID);
				// validate account balances
				Map<String, Integer> balances = new HashMap<String, Integer>();
				balances.put(assetID, 1000);
				TestUtils.validateAccountBalance(c, rcvrID, balances);
				return 1;
			},
			() -> {
				String txID = TestUtils.transact(c, assetID, sndrID, secondAddr, 1000);
				TestUtils.waitForPropagation(sc, txID);
				System.out.printf("\tID=%s\n", txID);
				// validate account balances
				Map<String, Integer> balances = new HashMap<String, Integer>();
				balances.put(assetID, 1000);
				TestUtils.validateAccountBalance(sc, secondRcvrID, balances);
				return 1;
			}
		);
		List<Future<Integer>> results = pool.invokeAll(txs);
		pool.shutdown();

		// validate transactions
		int success = 0;

		// counts number of successful spends
		for (Future<Integer> result : results) {
			try {
				success += result.get();
			} catch (Exception e) {
				// Transaction failed, but the actual cause is wrapped in an
				// ExecutionException. We expect the nested exception to be an
				// APIException with error code CH733 (Insufficient funds).
				// Any other exception should be bubbled up.
				Throwable nested = e.getCause();
				if (nested instanceof APIException) {
					APIException ex = (APIException) nested;
					if (!"CH733".equals(ex.code)) {
						throw e;
					}
				} else {
					throw e;
				}
			}
		}

		// validate the number of successful spends
		assert success != 0 : "Both transactions failed.";
		assert success != 2 : "Executed a double spend.";
		assert success == 1 : "Unexpected error.";
		System.out.println("\tsecond spend was unsuccessful.");
		return true;
	}
}
