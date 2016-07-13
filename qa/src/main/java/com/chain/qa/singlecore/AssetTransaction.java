package com.chain.qa.singlecore;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.List;

import com.chain.*;
import com.chain.qa.*;

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
		c = client;
		projectID = pID;
		issuerID = TestUtils.createIssuer(c, projectID, "Transaction");
		managerID = TestUtils.createManager(c, projectID, "Transaction");
		assetID = TestUtils.createAsset(c, issuerID, "Transaction");
		secondIssuerID = TestUtils.createIssuer(c, projectID, "Transaction Second");
		secondManagerID = TestUtils.createManager(c, projectID, "Transaction Second");
		secondAssetID = TestUtils.createAsset(c, secondIssuerID, "Transaction Second");
		assert testOneWayTransaction();
		assert testAtomicSwap();
		assert testIssueAndTransaction();
		assert test1of2Transaction();
		assert test2of2Transaction();
		assert testInsufficientSignatures();
	}

	/**
	 * Executes a one-way transaction and validates account balances.
	 */
	private static boolean testOneWayTransaction() throws Exception {
		String sndrID = TestUtils.createAccount(c, managerID, "One-way Transaction Sender");
		String rcvrID = TestUtils.createAccount(c, managerID, "One-way Transaction Receiver");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);

		// issue 1000 units of asset to sender
		TestUtils.issue(c, assetID, sndrID, 1000);

		// send 600 units of asset from sender to receiver
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(600));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(600));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse resp = c.submitTransaction(tx);
		System.out.printf("Executed a one-way transaction. ID=%s\n", resp.transactionID);

		// validate sender balance
		Map<String, Integer> balances = new HashMap<>();
		balances.put(assetID, 400);
		TestUtils.validateAccountBalance(c, sndrID, balances);
		// validate receiver balance
		balances = new HashMap<>();
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
		String sndrID = TestUtils.createAccount(c, managerID, "Atomic Swap Account A");
		String sndrAddr = TestUtils.createAddress(c, sndrID);
		String rcvrID = TestUtils.createAccount(c, secondManagerID, "Atomic Swap Account B");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);

		// issue 1000 units of the first asset to sndr
		TestUtils.issue(c, assetID, sndrID, 1000);

		// issue 1000 units of the second asset to rcvr
		TestUtils.issue(c, secondAssetID, rcvrID, 1000);

		// build first part of transaction
		// send 750 units of asset from sndr to rcvr
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(750));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(750));
		Transactor.Transaction partialTx = c.buildTransaction(build);

		// build second part of transaction
		// send 250 units of second asset from rcvr to sndr
		List<Transactor.BuildRequest.Input> inputs = new ArrayList<>();
		List<Transactor.BuildRequest.Output> outputs = new ArrayList<>();
		build = new Transactor.BuildRequest(partialTx, inputs, outputs);
		build.addInput(secondAssetID, rcvrID, BigInteger.valueOf(250));
		build.addAddressOutput(secondAssetID, sndrAddr, BigInteger.valueOf(250));
		Transactor.Transaction fullTx = c.buildTransaction(build);
		c.signTransaction(fullTx);
		Transactor.SubmitResponse resp = c.submitTransaction(fullTx);
		System.out.printf("Executed an atomic swap transaction. ID=%s\n", resp.transactionID);

		// validate sndr balances
		Map<String, Integer> balances = new HashMap<>();
		balances.put(assetID, 250);
		balances.put(secondAssetID, 250);
		TestUtils.validateAccountBalance(c, sndrID, balances);
		// validate rcvr balances
		balances = new HashMap<>();
		balances.put(assetID, 750);
		balances.put(secondAssetID, 750);
		TestUtils.validateAccountBalance(c, rcvrID, balances);
		return true;
	}

	/**
	 * Executes simultaneous issue/transaction and validates account balances.
	 */
	private static boolean testIssueAndTransaction() throws Exception {
		String sndrID = TestUtils.createAccount(c, managerID, "Issue/transaction Account A");
		String sndrAddr = TestUtils.createAddress(c, sndrID);
		String rcvrID = TestUtils.createAccount(c, managerID, "Issue/transaction Account B");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);

		// issue 500 units of asset to each account
		TestUtils.issue(c, assetID, sndrID, 500);
		TestUtils.issue(c, secondAssetID, rcvrID, 500);

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
		List<Transactor.BuildRequest.Input> inputs = new ArrayList<>();
		List<Transactor.BuildRequest.Output> outputs = new ArrayList<>();
		build = new Transactor.BuildRequest(partialTx, inputs, outputs);
		// issue 500 units of secondAsset to rcvr
		build.addIssueInput(secondAssetID);
		build.addAddressOutput(secondAssetID, rcvrAddr, BigInteger.valueOf(500));
		// send 500 units of asset from rcvr to sndr
		build.addInput(secondAssetID, rcvrID, BigInteger.valueOf(500));
		build.addAddressOutput(secondAssetID, sndrAddr, BigInteger.valueOf(500));
		Transactor.Transaction fullTx = c.buildTransaction(build);
		c.signTransaction(fullTx);
		Transactor.SubmitResponse resp = c.submitTransaction(fullTx);
		System.out.printf("Executed a simultaneous issue/transaction. ID=%s\n", resp.transactionID);

		// validate sndr balances
		Map<String, Integer> balances = new HashMap<>();
		balances.put(assetID, 500);
		balances.put(secondAssetID, 500);
		TestUtils.validateAccountBalance(c, sndrID, balances);
		// validate rcvr balances
		balances = new HashMap<>();
		balances.put(assetID, 500);
		balances.put(secondAssetID, 500);
		TestUtils.validateAccountBalance(c, rcvrID, balances);
		return true;
	}

	/**
	 * Executes a transaction involving a manager configured for 1 of 2 signatures required
     */
	private static boolean test1of2Transaction() throws Exception {
		// create manager
		List<AccountManager.CreateRequest.Key> keys = Arrays.asList(
			AccountManager.CreateRequest.Key.Generated(),
			AccountManager.CreateRequest.Key.Generated()
		);
		AccountManager.CreateRequest req = new AccountManager.CreateRequest("1 of 2 Manager", 1, keys);
		AccountManager mgr = c.createAccountManager(projectID, req);

		// setup
		String managerID = mgr.ID;
		String sndrID = TestUtils.createAccount(c, managerID, "1 of 2 Transaction Sender");
		String rcvrID = TestUtils.createAccount(c, managerID, "1 of 2 Transaction Receiver");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);
		String assetID = TestUtils.createAsset(c, issuerID, "1 of 2 Transaction");
		TestUtils.issue(c, assetID, sndrID, 1000);

		// submit transaction using first key
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(600));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(600));
		Transactor.Transaction tx = c.buildTransaction(build);
		Key.Store testStore = new Key.Store();
		testStore.add(new XPrvKey(mgr.keys[0].xprv, true));
		c.setSigner(new MemorySigner(testStore));
		c.signTransaction(tx);
		Transactor.SubmitResponse resp = c.submitTransaction(tx);
		System.out.printf("Transacted from manager configured for 1 of 2 signatures (using first key). ID=%s\n", resp.transactionID);

		// submit transaction using second key
		build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(300));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(300));
		tx = c.buildTransaction(build);
		testStore = new Key.Store();
		testStore.add(new XPrvKey(mgr.keys[1].xprv, true));
		c.signTransaction(tx);
		resp = c.submitTransaction(tx);
		System.out.printf("Transacted from manager configured for 1 of 2 signatures (using second key). ID=%s\n", resp.transactionID);
		return true;
	}

	/**
	 * Executes a transaction involving a manager configured for 2 of 2 signatures required
	 */
	private static boolean test2of2Transaction() throws Exception {
		// create manager
		List<AccountManager.CreateRequest.Key> keys = Arrays.asList(
			AccountManager.CreateRequest.Key.Generated(),
			AccountManager.CreateRequest.Key.Generated()
		);
		AccountManager.CreateRequest req = new AccountManager.CreateRequest("2 of 2 Manager", 2, keys);
		AccountManager mgr = c.createAccountManager(projectID, req);
		c.getKeyStore().add(new XPrvKey(mgr.keys[0].xprv, true));
		c.getKeyStore().add(new XPrvKey(mgr.keys[1].xprv, true));
		c.setSigner(new MemorySigner(c.getKeyStore()));

		// setup
		String managerID = mgr.ID;
		String sndrID = TestUtils.createAccount(c, managerID, "2 of 2 Transaction Sender");
		String rcvrID = TestUtils.createAccount(c, managerID, "2 of 2 Transaction Receiver");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);
		String assetID = TestUtils.createAsset(c, issuerID, "2 of 2 Transaction");
		TestUtils.issue(c, assetID, sndrID, 1000);

		// send 600 units of asset from sender to receiver
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(600));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(600));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse resp = c.submitTransaction(tx);
		System.out.printf("Transacted from manager configured for 2 of 2 signatures. ID=%s\n", resp.transactionID);
		return true;
	}

	/**
	 * Attempts to submit a transaction with an insufficient amount of signatures
	 */
	private static boolean testInsufficientSignatures() throws Exception {
		// create manager
		List<AccountManager.CreateRequest.Key> keys = Arrays.asList(
			AccountManager.CreateRequest.Key.Generated(),
			AccountManager.CreateRequest.Key.Generated()
		);
		AccountManager.CreateRequest req = new AccountManager.CreateRequest("2 of 2 Manager", 2, keys);
		AccountManager mgr = c.createAccountManager(projectID, req);

		// setup
		String managerID = mgr.ID;
		String sndrID = TestUtils.createAccount(c, managerID, "2 of 2 Transaction Sender");
		String rcvrID = TestUtils.createAccount(c, managerID, "2 of 2 Transaction Receiver");
		String rcvrAddr = TestUtils.createAddress(c, rcvrID);
		String assetID = TestUtils.createAsset(c, issuerID, "2 of 2 Transaction");
		TestUtils.issue(c, assetID, sndrID, 1000);

		// attempt to submit an unsigned tx
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(600));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(600));
		Transactor.Transaction tx = c.buildTransaction(build);
		try {
			c.submitTransaction(tx);
		} catch (APIException e) {
			if ("CH755".equals(e.code)) {
				c.cancelReservation(tx);
				System.out.printf("Failed to submit a tx with 0 signatures requiring 2.\n");
			} else {
				throw e;
			}
		}

		// attempt to submit a tx with insufficient signatures
		build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(600));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(600));
		tx = c.buildTransaction(build);
		c.getKeyStore().add(new XPrvKey(mgr.keys[0].xprv, true));
		c.setSigner(new MemorySigner(c.getKeyStore()));
		c.signTransaction(tx);
		try {
			c.submitTransaction(tx);
		} catch (APIException e) {
			if ("CH755".equals(e.code)) {
				System.out.printf("Failed to submit a tx with 1 signature requiring 2.\n");
			} else {
				throw e;
			}
		}
		return true;
	}
}
