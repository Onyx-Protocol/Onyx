package com.chain.qa.singlecore;

import java.math.BigInteger;
import java.util.HashMap;
import java.util.Map;

import com.chain.*;
import com.chain.qa.*;

/**
 * AssetIssuance tests different methods of issuing assets.
 */
public class AssetIssuance {
	private static TestClient c;
	private static String projectID;
	private static String issuerID;
	private static String managerID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID) throws Exception {
		c = client;
		projectID = pID;
		issuerID = TestUtils.createIssuer(c, projectID, "Issuance");
		managerID = TestUtils.createManager(c, projectID, "Issuance");
		assert testIssueToAccount();
		assert testIssueToAddress();
	}

	/**
	 * Issues 1000 units of an asset to an account ID and validates issued amounts.
	 */
	private static boolean testIssueToAccount() throws Exception {
		String assetID = TestUtils.createAsset(c, issuerID, "Account ID Issued");
		String acctID = TestUtils.createAccount(c, managerID, "Account ID Issued");

		// execute issuance
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addIssueInput(assetID, BigInteger.valueOf(1000));
		build.addAccountOutput(assetID, acctID, BigInteger.valueOf(1000));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse issue = c.submitTransaction(tx);
		System.out.printf("Issued to an account. ID=%s\n", issue.transactionID);

		// validate issuance and account balances
		TestUtils.validateAssetIssuance(c, assetID, 1000);
		Map<String, Integer> balances = new HashMap<>();
		balances.put(assetID, 1000);
		TestUtils.validateAccountBalance(c, acctID, balances);
		return true;
	}

	/**
	 * Issues 1000 units of an asset to an address and validates issued amounts.
	 */
	private static boolean testIssueToAddress() throws Exception {
		String acctID = TestUtils.createAccount(c, managerID, "Address Issued");
		String addr = TestUtils.createAddress(c, acctID);
		String assetID = TestUtils.createAsset(c, issuerID, "Address Issued");

		// execute issuance
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addIssueInput(assetID, BigInteger.valueOf(1000));
		build.addAddressOutput(assetID, addr, BigInteger.valueOf(1000));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse issue = c.submitTransaction(tx);
		System.out.printf("Issued to an address. ID=%s\n", issue.transactionID);

		// validate issuance and account balance
		TestUtils.validateAssetIssuance(c, assetID, 1000);
		Map<String, Integer> balances = new HashMap<>();
		balances.put(assetID, 1000);
		TestUtils.validateAccountBalance(c, acctID, balances);
		return true;
	}
}
