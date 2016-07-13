package com.chain.qa.singlecore;

import java.net.URL;

import com.chain.qa.*;

public class Main {
	public static void main(String [] args)
	throws Exception {
		System.out.println("Single core tests:");
		TestClient client = new TestClient(new URL(System.getenv("CHAIN_API_URL")));
		String project = TestUtils.createProject(client, "Single Core QA Tests");
		AssetIssuerCreation.runTests(client, project);
		AccountManagerCreation.runTests(client, project);
		AccountCreation.runTests(client, project);
		AddressCreation.runTests(client, project);
		AssetCreation.runTests(client, project);
		AssetIssuance.runTests(client, project);
		AssetTransaction.runTests(client, project);
		AssetRetirement.runTests(client, project);
		System.out.println("Finished.");
	}
}
