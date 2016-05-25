package chain.qa.baseline.singlecore;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;
import java.util.List;

import chain.qa.*;

import com.chain.*;

/**
 * AssetTransaction tests different methods of transacting assets between
 * accounts on a single core.
 */
public class AssetTransaction {
	private static TestClient c;
	private static String projectID;
	private static String issuerID;
	private static String managerID;
	private static String assetID;
	private static String secondIssuerID;
	private static String secondManagerID;
	private static String secondAssetID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID) throws Exception {
		// setup
		c = client;
		projectID = pID;
		issuerID = TestUtils.createIssuer(c, projectID, "Transaction");
		managerID = TestUtils.createManager(c, projectID, "Transaction");
		assetID = TestUtils.createAsset(c, issuerID, "Transaction");
		secondIssuerID = TestUtils.createIssuer(c, projectID, "Transaction Second");
		secondManagerID = TestUtils.createManager(c, projectID, "Transaction Second");
		secondAssetID = TestUtils.createAsset(c, secondIssuerID, "Transaction Second");

		// assertions
		assert testOneWayTransaction();
		assert testAtomicSwap();
		assert testIssueAndTransaction();
	}

	/**
	 * Executes a one-way transaction and validates account balances.
	 */
	private static boolean testOneWayTransaction() throws Exception {
		// setup
		String sndrID = TestUtils.createAccount(c, managerID, "One-way Transaction Sender");
		String rcvrID = TestUtils.createAccount(c, managerID, "One-way Transaction Receiver");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);

		// issue 1000 units of asset to sender
		String issueID = TestUtils.issue(c, assetID, sndrID, 1000);

		// send 600 units of asset from sender to receiver
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(600));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(600));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse resp = c.submitTransaction(tx);

		System.out.printf("Executed a one-way transaction. ID=%s\n", resp.transactionID);

		// validate sender balance
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 400);
		TestUtils.validateAccountBalance(c, sndrID, balances);

		// validate receiver balance
		balances = new HashMap<String, Integer>();
		balances.put(assetID, 600);
		TestUtils.validateAccountBalance(c, rcvrID, balances);
		return true;
	}

	/**
	 * Executes an atomic swap transaction and validates account balances.
	 * Each account is created on separate managers and issued assets from
	 * separate issuers.
	 */
	private static boolean testAtomicSwap() throws Exception {
		// setup
		String sndrID = TestUtils.createAccount(c, managerID, "Atomic Swap Account A");
		String sndrAddr = TestUtils.createAddress(c, sndrID);
		String rcvrID = TestUtils.createAccount(c, secondManagerID, "Atomic Swap Account B");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);

		// issue 1000 units of the first asset to sndr
		String issueID = TestUtils.issue(c, assetID, sndrID, 1000);

		// issue 1000 units of the second asset to rcvr
		issueID = TestUtils.issue(c, secondAssetID, rcvrID, 1000);

		// build first part of transaction
		// send 750 units of asset from sndr to rcvr
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(750));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(750));
		Transactor.Transaction partialTx = c.buildTransaction(build);

		// build second part of transaction
		// send 250 units of second asset from rcvr to sndr
		List<Transactor.BuildRequest.Input> inputs = new ArrayList<Transactor.BuildRequest.Input>();
		List<Transactor.BuildRequest.Output> outputs = new ArrayList<Transactor.BuildRequest.Output>();
		build = new Transactor.BuildRequest(partialTx, inputs, outputs);
		build.addInput(secondAssetID, rcvrID, BigInteger.valueOf(250));
		build.addAddressOutput(secondAssetID, sndrAddr, BigInteger.valueOf(250));
		Transactor.Transaction fullTx = c.buildTransaction(build);
		c.signTransaction(fullTx);
		Transactor.SubmitResponse resp = c.submitTransaction(fullTx);

		System.out.printf("Executed an atomic swap transaction. ID=%s\n", resp.transactionID);

		// validate sndr balances
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 250);
		balances.put(secondAssetID, 250);
		TestUtils.validateAccountBalance(c, sndrID, balances);

		// validate rcvr balances
		balances = new HashMap<String, Integer>();
		balances.put(assetID, 750);
		balances.put(secondAssetID, 750);
		TestUtils.validateAccountBalance(c, rcvrID, balances);
		return true;
	}

	/**
	 * Executes simultaneous issue/transaction and validates account balances.
	 */
	private static boolean testIssueAndTransaction() throws Exception {
		// setup
		String sndrID = TestUtils.createAccount(c, managerID, "Issue/transaction Account A");
		String sndrAddr = TestUtils.createAddress(c, sndrID);
		String rcvrID = TestUtils.createAccount(c, managerID, "Issue/transaction Account B");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);
		// issue 500 units of asset to each account
		String issueID = TestUtils.issue(c, assetID, sndrID, 500);
		issueID = TestUtils.issue(c, assetID, rcvrID, 500);

		// build first part of the transaction
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		// issue 500 units of asset to sndr
		build.addIssueInput(assetID);
		build.addAddressOutput(assetID, sndrAddr, BigInteger.valueOf(500));
		// send 500 units of asset from sndr to rcvr
		build.addInput(assetID, sndrID, BigInteger.valueOf(500));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(500));
		Transactor.Transaction partialTx = c.buildTransaction(build);

		// build second part of the transaction
		List<Transactor.BuildRequest.Input> inputs = new ArrayList<Transactor.BuildRequest.Input>();
		List<Transactor.BuildRequest.Output> outputs = new ArrayList<Transactor.BuildRequest.Output>();
		build = new Transactor.BuildRequest(partialTx, inputs, outputs);
		// issue 500 units of asset to rcvr
		build.addIssueInput(assetID);
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(500));
		// send 500 units of asset from rcvr to sndr
		build.addInput(assetID, rcvrID, BigInteger.valueOf(500));
		build.addAddressOutput(assetID, sndrAddr, BigInteger.valueOf(500));
		Transactor.Transaction fullTx = c.buildTransaction(build);
		c.signTransaction(fullTx);
		Transactor.SubmitResponse resp = c.submitTransaction(fullTx);

		System.out.printf("Executed a simultaneous issue/transaction. ID=%s\n", resp.transactionID);

		// validate sndr balances
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 1000);
		TestUtils.validateAccountBalance(c, sndrID, balances);

		// validate rcvr balances
		balances = new HashMap<String, Integer>();
		balances.put(assetID, 1000);
		TestUtils.validateAccountBalance(c, rcvrID, balances);
		return true;
	}
}
