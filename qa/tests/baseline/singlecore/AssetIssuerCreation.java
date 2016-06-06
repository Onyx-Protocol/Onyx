package chain.qa.baseline.singlecore;

import java.util.ArrayList;
import java.util.List;

import chain.qa.*;

import com.chain.*;

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
		// setup
		c = client;
		projectID = pID;

		// assertions
		assert testGeneratedKeys();
		assert testProvidedKeys();
	}

	/**
	 * Creates a basic issuer with generated keys and validates its properties.
	 */
	private static boolean testGeneratedKeys()
	throws ChainException {
		// create issuer with generated keys
		String label = "Key Generated";
		List<IssuerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(IssuerNode.CreateRequest.Key.Generated());
		IssuerNode.CreateRequest req = new IssuerNode.CreateRequest(label, 1, keys);
		IssuerNode isr = c.createIssuerNode(projectID, req);

		System.out.printf("Created issuer with generated keys. ID=%s\n", isr.ID);

		// validate issuer with generated keys
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
		// create issuer with provided keys
		String label = "Key Provided";
		String xpub = TestUtils.XPUB[0];
		List <IssuerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(IssuerNode.CreateRequest.Key.XPub(xpub));
		IssuerNode.CreateRequest req = new IssuerNode.CreateRequest(label, 1, keys);
		IssuerNode isr = c.createIssuerNode(projectID, req);

		System.out.printf("Created issuer with provided keys. ID=%s\n", isr.ID);

		// validate issuer with provided keys
		int sigsReq = isr.signaturesRequired.intValue();
		assert isr.ID != null : "ID should not equal null.";
		assert isr.label.equals(label) : TestUtils.fail("label", isr.label, label);
		assert isr.keys[0].xpub != null : "xpub should not equal null.";
		assert isr.keys[0].xpub.equals(xpub) : TestUtils.fail("xpub", isr.keys[0].xpub, xpub);
		assert sigsReq == 1 : TestUtils.fail("sigs required", sigsReq, 1);
		return true;
	}
}
