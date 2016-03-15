package chain.qa.baseline;

import java.net.URL;

import chain.qa.TestClient;

import com.chain.*;

public class Main {
	public static void main(String [] args) throws ChainException, Exception {
		System.out.println("running baseline test");
		URL url = new URL(args[0]);
		TestClient client = new TestClient(url);
		BasicIssue.run(client);
		System.out.println("finished");
	}
}
