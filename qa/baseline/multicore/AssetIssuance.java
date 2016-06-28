package chain.qa.baseline.multicore;

import java.util.HashMap;
import java.util.Map;
import java.util.List;

import com.chain.*;

import chain.qa.*;

/**
 * AssetIssuance tests asset issuance in a multi-core network.
 */
public class AssetIssuance {
	// first core
	private static TestClient c;
	private static String issuerID;
	private static String managerID;
	// second core
	private static TestClient sc;
	private static String secondManagerID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID, TestClient secondClient, String spID)
	throws Exception {
		// setup first core
		c = client;
		issuerID = TestUtils.createIssuer(c, pID, "Issuance");
		managerID = TestUtils.createManager(c, pID, "Issuance");

		// setup second core
		sc = secondClient;
		secondManagerID = TestUtils.createManager(sc, spID, "Issuance Second");

		// assertions
		assert testCrossCoreIssue();
	}

	/**
	 * Creates an asset on the first core and issues it to an account on
	 * the second core. The assertions check that the first core's issuer
	 * and the second's manager both account for the issuance. Also checks both
	 * cores can lookup the asset definition.
	 */
	public static boolean testCrossCoreIssue()
	throws Exception {
		// create asset
		String label = "Cross Core Asset";
		HashMap<String, Object> def = new HashMap<String, Object>();
		def.put("Asset", "Definition");
		String assetID = TestUtils.createAsset(c, issuerID, label, def);

		// issue 1000 units of asset to second core
		String acctID = TestUtils.createAccount(sc, secondManagerID, "Cross Core Account");
		String addr = TestUtils.createAddress(sc, acctID);
		String txID = TestUtils.issueToAddress(c, assetID, addr, 1000);
		TestUtils.waitForPropagation(sc, txID);
		System.out.printf("Issued cross core. ID=%s\n", txID);

		// asset definition represented as json string with whitespace stripped
		String defCheck = "{\"Asset\":\"Definition\"}";

		// validate the first core can lookup asset
		TestUtils.validateAssetDefinition(c, assetID, defCheck);

		// validate the first core's issuer accounts for the issuance
		TestUtils.validateAssetIssuance(c, assetID, 1000);

		// validate the second core can lookup asset
		TestUtils.validateAssetDefinition(sc, assetID, defCheck);

		// validate the second core's manager accounts for the issuance
		Map<String, Integer> balances = new HashMap<String, Integer>();
		balances.put(assetID, 1000);
		TestUtils.validateAccountBalance(sc, acctID, balances);
		return true;
	}
}
