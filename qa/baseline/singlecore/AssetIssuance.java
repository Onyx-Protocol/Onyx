package chain.qa.baseline.singlecore;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;
import java.util.List;

import chain.qa.*;

import com.chain.*;

/**
 * AssetIssuance tests different methods of issuing assets.
 */
public class AssetIssuance {
	private static TestClient c;
	private static String projectID;
	private static String issuerID;
	private static String managerID;
	private static String acctID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID) throws Exception {
		// setup
		c = client;
		projectID = pID;
		issuerID = TestUtils.createIssuer(c, projectID, "Issuance");
		managerID = TestUtils.createManager(c, projectID, "Issuance");

		// assertions
		assert testIssueToAccount();
		assert testIssueToAddress();
	}

	/**
	 * Issues 1000 units of an asset to an account ID and validates issued amounts.
	 */
	private static boolean testIssueToAccount() throws Exception {
		// create asset
		String assetID = TestUtils.createAsset(c, issuerID, "Account ID Issued");
		String acctID = TestUtils.createAccount(c, managerID, "Account ID Issued");

		// issue asset using an account ID
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addIssueInput(assetID, BigInteger.valueOf(1000));
		build.addAccountOutput(assetID, acctID, BigInteger.valueOf(1000));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse issue = c.submitTransaction(tx);

		System.out.printf("Issued to an account. ID=%s\n", issue.transactionID);

		// validate issuance
		TestUtils.validateAssetIssuance(c, assetID, 1000);

		// validate account balance
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 1000);
		TestUtils.validateAccountBalance(c, acctID, balances);
		return true;
	}

	/**
	 * Issues 1000 units of an asset to an address and validates issued amounts.
	 */
	private static boolean testIssueToAddress() throws Exception {
		// create asset
		String acctID = TestUtils.createAccount(c, managerID, "Address Issued");
		String addr = TestUtils.createAddress(c, acctID);
		String assetID = TestUtils.createAsset(c, issuerID, "Address Issued");

		// issue assetID using an address
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addIssueInput(assetID, BigInteger.valueOf(1000));
		build.addAddressOutput(assetID, addr, BigInteger.valueOf(1000));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse issue = c.submitTransaction(tx);

		System.out.printf("Issued to an address. ID=%s\n", issue.transactionID);

		// validate issuance
		TestUtils.validateAssetIssuance(c, assetID, 1000);

		// validate account balance
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 1000);
		TestUtils.validateAccountBalance(c, acctID, balances);
		return true;
	}
}
