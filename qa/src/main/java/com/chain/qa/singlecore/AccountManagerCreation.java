package com.chain.qa.singlecore;

import java.util.ArrayList;
import java.util.List;

import com.chain.*;
import com.chain.qa.*;


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
		c = client;
		projectID = pID;
		assert testGeneratedKeys();
		assert testProvidedKeys();
		assert testAccountProvidedKeys();
	}

	/**
	 * Creates a basic manager with generated keys and validates its properties.
	 */
	private static boolean testGeneratedKeys()
	throws ChainException {
		String label = "Key Generated";
		List<AccountManager.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(AccountManager.CreateRequest.Key.Generated());
		AccountManager.CreateRequest req = new AccountManager.CreateRequest(label, 1, keys);
		AccountManager mgr = c.createAccountManager(projectID, req);
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
		String label = "Key Provided";
		String xpub = TestUtils.XPUB[0];
		List<AccountManager.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(AccountManager.CreateRequest.Key.XPub(xpub));
		AccountManager.CreateRequest req = new AccountManager.CreateRequest(label, 1, keys);
		AccountManager mgr = c.createAccountManager(projectID, req);
		System.out.printf("Created manager with provided keys. ID=%s\n", mgr.ID);

		// validate manager
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
		String label = "Account Key";
		String xpub = TestUtils.XPUB[0];
		List<AccountManager.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(AccountManager.CreateRequest.Key.Account());
		AccountManager.CreateRequest req = new AccountManager.CreateRequest(label, 1, keys);
		AccountManager mgr = c.createAccountManager(projectID, req);
		System.out.printf("Created manager with account provided keys. ID=%s\n", mgr.ID);

		// validate manager
		int sigsReq = mgr.signaturesRequired.intValue();
		assert mgr.ID != null : "ID should not equal null.";
		assert mgr.label.equals(label) : TestUtils.fail("label", mgr.label, label);
		assert mgr.keys[0].xpub == null : "xpub should equal null.";
		assert mgr.keys[0].xprv == null : "xprv should equal null.";
		assert sigsReq == 1 : TestUtils.fail("sigs required", sigsReq, 1);
		return true;
	}
}
