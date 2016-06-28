package chain.qa.baseline.singlecore;

import java.util.ArrayList;
import java.util.List;

import chain.qa.*;

import com.chain.*;

/**
 * AccountManagerCreation tests different setup configurations for managers
 * connected to a single core.
 */
public class AccountManagerCreation {
	private static TestClient c;
	private static String projectID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID)
	throws ChainException {
		// setup
		c = client;
		projectID = pID;

		// assertions
		assert testGeneratedKeys();
		assert testProvidedKeys();
		assert testAccountProvidedKeys();
	}

	/**
	 * Creates a basic manager with generated keys and validates its properties.
	 */
	private static boolean testGeneratedKeys()
	throws ChainException {
		// create manager with generated keys
		String label = "Key Generated";
		List<ManagerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(ManagerNode.CreateRequest.Key.Generated());
		ManagerNode.CreateRequest req = new ManagerNode.CreateRequest(label, 1, keys);
		ManagerNode mgr = c.createManagerNode(projectID, req);

		System.out.printf("Created manager with generated keys. ID=%s\n", mgr.ID);

		// validate manager with generated keys
		int sigsReq = mgr.signaturesRequired.intValue();
		assert mgr.ID != null : "ID should not equal null.";
		assert mgr.label.equals(label) : TestUtils.fail("label", mgr.label, label);
		assert mgr.keys[0].xpub != null : "xpub should not equal null.";
		assert mgr.keys[0].xprv != null : "xprv should not equal null.";
		assert sigsReq == 1 : TestUtils.fail("sigs required", sigsReq, 1);
		return true;
	}

	/**
	 * Creates a basic manager with provided keys and validates its properties.
	 */
	private static boolean testProvidedKeys()
	throws ChainException {
		// create manager with provided keys
		String label = "Key Provided";
		String xpub = TestUtils.XPUB[0];
		List<ManagerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(ManagerNode.CreateRequest.Key.XPub(xpub));
		ManagerNode.CreateRequest req = new ManagerNode.CreateRequest(label, 1, keys);
		ManagerNode mgr = c.createManagerNode(projectID, req);

		System.out.printf("Created manager with provided keys. ID=%s\n", mgr.ID);

		// validate manager with provided keys
		int sigsReq = mgr.signaturesRequired.intValue();
		assert mgr.ID != null : "ID should not equal null.";
		assert mgr.label.equals(label) : TestUtils.fail("label", mgr.label, label);
		assert mgr.keys[0].xpub != null : "xpub should not equal null.";
		assert mgr.keys[0].xpub.equals(xpub) : TestUtils.fail("xpub", mgr.keys[0].xpub, xpub);
		assert sigsReq == 1 : TestUtils.fail("sigs required", sigsReq, 1);
		return true;
	}

	/**
	 * Creates a basic manager with account provided keys
	 * and validates its properties.
	 */
	private static boolean testAccountProvidedKeys()
	throws ChainException {
		// create manager with provided keys
		String label = "Account Key";
		String xpub = TestUtils.XPUB[0];
		List<ManagerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(ManagerNode.CreateRequest.Key.Account());
		ManagerNode.CreateRequest req = new ManagerNode.CreateRequest(label, 1, keys);
		ManagerNode mgr = c.createManagerNode(projectID, req);

		System.out.printf("Created manager with account provided keys. ID=%s\n", mgr.ID);

		// validate manager with provided keys
		int sigsReq = mgr.signaturesRequired.intValue();
		assert mgr.ID != null : "ID should not equal null.";
		assert mgr.label.equals(label) : TestUtils.fail("label", mgr.label, label);
		assert mgr.keys[0].xpub == null : "xpub should equal null.";
		assert mgr.keys[0].xprv == null : "xprv should equal null.";
		assert sigsReq == 1 : TestUtils.fail("sigs required", sigsReq, 1);
		return true;
	}
}
