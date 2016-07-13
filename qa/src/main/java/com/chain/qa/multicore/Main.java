package com.chain.qa.multicore;

import java.net.URL;

import com.chain.qa.*;

import com.chain.*;

public class Main {
	public static void main(String [] args)
	throws Exception {
		System.out.println("Multi-core tests:");
		TestClient client = new TestClient(new URL(System.getenv("CHAIN_API_URL")));
		TestClient secondClient = new TestClient(new URL(System.getenv("SECOND_API_URL")));
		String project = TestUtils.createProject(client, "QA Test");
		String secondProject = TestUtils.createProject(secondClient, "QA Test");
		AssetIssuance.runTests(client, project, secondClient, secondProject);
		AssetTransaction.runTests(client, project, secondClient, secondProject);
		System.out.println("finished");
	}
}
