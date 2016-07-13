package com.chain.qa.singlecore;

import java.util.HashMap;

import com.chain.*;
import com.chain.qa.*;

/**
 * AssetCreation tests the creation of assets.
 */
public class AssetCreation {
	private static TestClient c;
	private static String projectID;
	private static String issuerID;
	private static String managerID;
	private static String acctID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID)
	throws Exception {
		c = client;
		projectID = pID;
		issuerID = TestUtils.createIssuer(c, projectID, "Asset Creation");
		managerID = TestUtils.createManager(c, projectID, "Asset Creation");
		acctID = TestUtils.createAccount(c, managerID, "Asset Creation");
		assert testAssetCreation();
		assert testAssetCreationWithDefinition();
	}

	/**
	 * Creates basic asset and validates its properties.
	 */
	private static boolean testAssetCreation()
	throws ChainException {
		String label = "Asset w/o Definition";
		Asset asset = c.createAsset(issuerID, label);
		System.out.printf("Created an asset. ID=%s\n", asset.ID);
		assert asset.ID != null : "ID should not equal null.";
		assert asset.label.equals(label) : TestUtils.fail("label", asset.label, label);
		return true;
	}

	/**
	 * Creates asset w/ defintion, issues asset and validates its
	 * properties and definition from the blockchain.
	 */
	private static boolean testAssetCreationWithDefinition()
	throws Exception {
		// create asset w/ definition and issue
		String label = "Asset w/ Definition";
		HashMap<String, Object> def = new HashMap<>();
		def.put("Asset", "Definition");
		Asset asset = c.createAsset(issuerID, label, def);
		TestUtils.issue(c, asset.ID, acctID, 1000);
		System.out.printf("Created an asset with a definition. ID=%s\n", asset.ID);

		// validate asset
		assert asset.ID != null : "ID should not equal null.";
		assert asset.label.equals("Asset w/ Definition") : TestUtils.fail("label", asset.label, label);
		// validate core can lookup asset definition represented as json string with whitespace stripped
		String defCheck = "{\"Asset\":\"Definition\"}";
		TestUtils.validateAssetDefinition(c, asset.ID, defCheck);
		return true;
	}
}
