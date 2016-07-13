package com.chain.qa.singlecore;

import java.util.ArrayList;
import java.util.List;

import com.chain.*;
import com.chain.qa.*;

/**
 * AssetIssuerCreation tests different setup configurations for issuers
 * connected to a single core.
 */
public class AssetIssuerCreation {
	private static TestClient c;
	private static String projectID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID)
	throws Exception {
		c = client;
		projectID = pID;
		assert testGeneratedKeys();
		assert testProvidedKeys();
	}

	/**
	 * Creates a basic issuer with generated keys and validates its properties.
	 */
	private static boolean testGeneratedKeys()
	throws ChainException {
		String label = "Key Generated";
		List<AssetIssuer.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(AssetIssuer.CreateRequest.Key.Generated());
		AssetIssuer.CreateRequest req = new AssetIssuer.CreateRequest(label, 1, keys);
		AssetIssuer isr = c.createAssetIssuer(projectID, req);
		System.out.printf("Created issuer with generated keys. ID=%s\n", isr.ID);

		// validate issuer
		int sigsReq = isr.signaturesRequired.intValue();
		assert isr.ID != null : "ID should not equal null";
		assert isr.label.equals(label) : TestUtils.fail("label", isr.label, label);
		assert isr.keys[0].xpub != null : "xpub should not equal null.";
		assert isr.keys[0].xprv != null : "xprv should not equal null.";
		assert sigsReq == 1 : TestUtils.fail("sigs required", sigsReq, 1);
		return true;
	}

	/**
	 * Creates a basic issuer with provided keys and validates its properties.
	 */
	private static boolean testProvidedKeys()
	throws ChainException {
		String label = "Key Provided";
		String xpub = TestUtils.XPUB[0];
		List <AssetIssuer.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(AssetIssuer.CreateRequest.Key.XPub(xpub));
		AssetIssuer.CreateRequest req = new AssetIssuer.CreateRequest(label, 1, keys);
		AssetIssuer isr = c.createAssetIssuer(projectID, req);
		System.out.printf("Created issuer with provided keys. ID=%s\n", isr.ID);

		// validate issuer
		int sigsReq = isr.signaturesRequired.intValue();
		assert isr.ID != null : "ID should not equal null.";
		assert isr.label.equals(label) : TestUtils.fail("label", isr.label, label);
		assert isr.keys[0].xpub != null : "xpub should not equal null.";
		assert isr.keys[0].xpub.equals(xpub) : TestUtils.fail("xpub", isr.keys[0].xpub, xpub);
		assert sigsReq == 1 : TestUtils.fail("sigs required", sigsReq, 1);
		return true;
	}
}
