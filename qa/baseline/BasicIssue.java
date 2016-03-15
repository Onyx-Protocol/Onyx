package chain.qa.baseline;

import chain.qa.*;

public class BasicIssue {
	public static void run(TestClient client) throws Exception {
		String projectID = TestUtils.createProject(client, "BasicIssue", "QA Project");
		String issuerID = TestUtils.createIssuerNode(client, "BasicIssue", projectID, "Issuer");
		String managerID = TestUtils.createManagerNode(client, "BasicIssue", projectID, "Manager");
		String assetID = TestUtils.createAsset(client, "BasicIssue", issuerID, "Asset");
		String accountID = TestUtils.createAccount(client, "BasicIssue", managerID, "Account");
		String txID = TestUtils.issue(client, "BasicIssue", issuerID, assetID, accountID, 1000);
	}
}
