package com.chain.qa.singlecore;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.List;

import com.chain.*;
import com.chain.qa.*;

/**
 * AssetRetirement tests asset retirement on a single core.
 */
public class AssetRetirement {
	private static TestClient c;
	private static String projectID;
	private static String issuerID;
	private static String managerID;
	private static String acctID;
	private static String assetID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID) throws Exception {
		c = client;
		projectID = pID;
		issuerID = TestUtils.createIssuer(c, projectID, "Asset Retirment");
		managerID = TestUtils.createManager(c, projectID, "Asset Retirment");
		assetID = TestUtils.createAsset(c, issuerID, "Asset Retirement");
		acctID = TestUtils.createAccount(c, managerID, "Asset Retirement");
		assert testBasicRetirement() : "asset amount should be unavailable.";
	}

	/**
	 * Retires 500 units of an asset and validates asset/acct balances
	 */
	private static boolean testBasicRetirement() throws Exception {
		TestUtils.issue(c, assetID, acctID, 1000);
		Transactor.BuildRequest.Input input = new Transactor.BuildRequest.Input(assetID, acctID, BigInteger.valueOf(400));
		Transactor.BuildRequest.Output output = Transactor.BuildRequest.Output.RetireOutput(assetID, BigInteger.valueOf(400));
		Transactor.BuildRequest build = new Transactor.BuildRequest(Arrays.asList(input), Arrays.asList(output));
		Transactor.Transaction tx = c.buildTransaction(build);
		c.signTransaction(tx);
		Transactor.SubmitResponse resp = c.submitTransaction(tx);
		System.out.printf("Retired an asset. ID=%s\n", resp.transactionID);

		// validate issued and retired amounts
		Asset check = c.getAsset(assetID);
		int total = check.issued.total.intValue();
		int confirmed = check.issued.confirmed.intValue();
		assert total == 1000 : TestUtils.fail("total", total, 1000);
		assert confirmed == 1000 : TestUtils.fail("confirmed", confirmed, 1000);

		// validate retired amount
		total = check.retired.total.intValue();
		confirmed = check.retired.confirmed.intValue();
		assert total == 400 : TestUtils.fail("total", total, 400);
		assert confirmed == 400 : TestUtils.fail("confirmed", confirmed, 400);

		// validate account balance
		Asset.BalancePage abp = c.listAccountBalances(acctID);
		List<Asset.Balance> acct = abp.balances;
		int size = acct.size();
		confirmed = acct.get(0).confirmed.intValue();
		assert size == 1 : TestUtils.fail("# of assets", size, 1);
		assert confirmed == 600 : TestUtils.fail("balance", confirmed, 600);

		// validate funds are not available
		String rcvrID = TestUtils.createAccount(c, managerID, "Asset Retirement Second");
		input = new Transactor.BuildRequest.Input(assetID, acctID, BigInteger.valueOf(601));
		output = new Transactor.BuildRequest.Output(assetID, rcvrID, null, BigInteger.valueOf(601));
		build = new Transactor.BuildRequest(Arrays.asList(input), Arrays.asList(output));
		try {
			c.transfer(build);
			// asset amount was not properly retired
			return false;
		} catch (APIException e) {
			if (!"CH743".equals(e.code)) {
				// a unexpected exception was thrown and should bubble up
				throw e;
			}
		}
		return true;
	}
}
