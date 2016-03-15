package chain.qa;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import com.chain.*;

public class TestUtils {
	public static String createProject(TestClient client, String test, String name) throws Exception {
		String projectID = client.createProject(name);
		System.out.printf("test=%s,function=%s,id=%s\n", test, "createProject", projectID);
		return projectID;
	}

	public static String createIssuerNode(Client client, String test, String projectID, String label) throws ChainException {
		List<IssuerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(new IssuerNode.CreateRequest.Key().Generated());
		IssuerNode.CreateRequest req = new IssuerNode.CreateRequest(label, 1, keys);
		IssuerNode in = client.createIssuerNode(projectID, req);
		client.getKeyStore().add(new XPrvKey(in.keys[0].xprv, true));
		System.out.printf("test=%s,function=%s,id=%s\n", test, "createIssuerNode", in.ID);
		return in.ID;
	}

	public static String createAsset(Client client, String test, String inID, String label) throws ChainException {
		Asset asset = client.createAsset(inID, label);
		System.out.printf("test=%s,function=%s,id=%s\n", test, "createIssuerNode", asset.ID);
		return asset.ID;
	}

	public static String createAsset(Client client, String test, String inID, String label, Map<String, Object> def) throws ChainException {
		Asset asset = client.createAsset(inID, label, def);
		System.out.printf("test=%s,function=%s,id=%s\n", test, "createIssuerNode", asset.ID);
		return asset.ID;
	}

	public static String issue(Client client, String test, String inID, String assetID, String accountID, Integer amount) throws ChainException {
		List<Asset.IssueOutput> outputs = new ArrayList<>();
		outputs.add(new Asset.IssueOutput(accountID, null, BigInteger.valueOf(amount)));
		Transactor.SubmitResponse resp = client.issue(assetID, outputs);
		System.out.printf("test=%s,function=%s,id=%s\n", test, "issue", resp.transactionID);
		return resp.transactionID;
	}

	public static String createManagerNode(Client client, String test, String projectID, String label) throws ChainException {
		List<ManagerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(ManagerNode.CreateRequest.Key.Generated());
		ManagerNode.CreateRequest req = new ManagerNode.CreateRequest(label, 1, keys);
		ManagerNode mn = client.createManagerNode(projectID, req);
		client.getKeyStore().add(new XPrvKey(mn.keys[0].xprv, true));
		System.out.printf("test=%s,function=%s,id=%s\n", test, "createManagerNode", mn.ID);
		return mn.ID;
	}

	public static String createAccount(Client client, String test, String mnID, String label) throws ChainException {
		Account.CreateRequest req = new Account.CreateRequest(label);
		Account account = client.createAccount(mnID, req);
		System.out.printf("test=%s,function=%s,id=%s\n", test, "createAccount", account.ID);
		return account.ID;
	}

	public static String createAddress(Client client, String test, String accountID) throws ChainException {
		Address address = client.createAddress(accountID);
		System.out.printf("test=%s,function=%s,id=%s\n", test, "createAddress", address.ID);
		return address.ID;
	}
}
