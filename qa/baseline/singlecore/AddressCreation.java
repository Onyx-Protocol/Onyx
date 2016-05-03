package chain.qa.baseline.singlecore;

import chain.qa.*;

import com.chain.*;

/**
 * AddressCreation tests the creation of account address.
 */
public class AddressCreation {
	private static TestClient c;
	private static String projectID;
	private static String managerID;
	private static String acctID;

	/**
	 * Runs tests
	 */
	public static void runTests(TestClient client, String pID)
	throws Exception {
		// setup
		c = client;
		projectID = pID;
		managerID = TestUtils.createManager(c, projectID, "Address Creation");
		acctID = TestUtils.createAccount(c, managerID, "Address Creation");

		// assertions
		assert testAddressCreation();
		assert testAddressDuplication();
	}

	/**
	 * Creates an account address and validates its properties.
	 */
	private static boolean testAddressCreation()
	throws ChainException {
		// create address
		Address addr = c.createAddress(acctID);

		System.out.printf("Created an account address. accountID=%s\n", acctID);

		// validate address
		assert addr.address != null : "address should not equal null.";
		return true;
	}

	/**
	 * Creates two account addresses and validates they are unique.
	 */
	private static boolean testAddressDuplication()
	throws ChainException {
		// create addresses
		Address addr1 = c.createAddress(acctID);
		Address addr2 = c.createAddress(acctID);

		System.out.printf("Created two account address. accountID=%s \n", acctID);

		// validate address
		assert !addr1.address.equals(addr2.address) : "addresses should be unique.";
		return true;
	}
}
