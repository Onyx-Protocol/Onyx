package com.chain.qa.singlecore;

import com.chain.*;
import com.chain.qa.*;

/**
 * AccountCreation tests the creation of accounts.
 */
public class AccountCreation {
	private static TestClient c;
	private static String projectID;
	private static String managerID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID)
	throws Exception {
		c = client;
		projectID = pID;
		managerID = TestUtils.createManager(c, projectID, "Account Creation");
		assert testAccountCreation();
	}

	/**
	 * Creates an account and validates its properties.
	 */
	private static boolean testAccountCreation()
	throws ChainException {
		String label = "Account Creation";
		Account acct = c.createAccount(managerID, label);
		System.out.printf("Created an account. ID=%s\n", acct.ID);
		assert acct.ID != null : "ID should not equal null.";
		assert acct.label.equals(label) : TestUtils.fail("label", acct.label, label);
		return true;
	}
}
