package chain.qa;

import java.util.concurrent.Callable;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.concurrent.TimeoutException;
import java.util.List;
import java.util.Map;

import com.chain.*;

/**
 * TestUtils wraps com.chain.Client methods and provides a simplified api for
 * testing. Each function takes a client object as one of its args.
 * The client will make the api call and the resulting object's ID will be
 * returned as a String.
 */
public class TestUtils {
	// generated xprv/xpub pairs for testing
	public static final String[] XPRV = {
		"xprv9wrB8xMKZBX5LPwVqPyRSbn8JWcoZWYwYpMhCG7rLy6C7yzMxLvTBKZ29gX75cUtFjgDzFpRPqrUsTezjM2A3bYds8ZpuQinWzLPQEUWVpJ"
	};
	public static final String[] XPUB = {
		"xpub6AqXYTtDPZ5NYt1xwRWRojirrYTHxyGnv3HHzeXTuJdAznKWVtEhj7sVzyMuJMn1E65uhw7pozjFsFaa4nRJBiDijr7do4zZ1CwM8TjTP3G"
	};

	/**
	 * RetireOutput is used to build asset retirement outputs.
	 */
	public static class RetireOutput extends Transactor.BuildRequest.Output {
		public String type;

		public RetireOutput(String assetID, BigInteger amount) {
			super(assetID, null, null, amount);
			this.type = "retire";
		}
	}

	/**
	 * Creates a project.
	 */
	public static String createProject(TestClient c, String name)
	throws Exception {
		return c.createProject(name);
	}

	/**
	 * Creates a 1 of 1 issuer, with generated keys, and adds its xprv
	 * to the Chain c key store.
	 */
	public static String createIssuer(Client c, String projID, String label)
	throws ChainException {
		List<IssuerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(IssuerNode.CreateRequest.Key.Generated());
		IssuerNode.CreateRequest req = new IssuerNode.CreateRequest(label, 1, keys);
		IssuerNode isr = c.createIssuerNode(projID, req);
		c.getKeyStore().add(new XPrvKey(isr.keys[0].xprv, true));
		c.setSigner(new MemorySigner(c.getKeyStore()));
		return isr.ID;
	}

	/**
	 * Creates a 1 of 1 manager, with generated keys, and adds its xprv
	 * to the Chain c key store.
	 */
	public static String createManager(Client c, String projID, String label)
	throws ChainException {
		List<ManagerNode.CreateRequest.Key> keys = new ArrayList<>();
		keys.add(ManagerNode.CreateRequest.Key.Generated());
		ManagerNode.CreateRequest req = new ManagerNode.CreateRequest(label, 1, keys);
		ManagerNode mgr = c.createManagerNode(projID, req);
		c.getKeyStore().add(new XPrvKey(mgr.keys[0].xprv, true));
		c.setSigner(new MemorySigner(c.getKeyStore()));
		return mgr.ID;
	}

	/**
	 * Creates an asset.
	 */
	public static String createAsset(Client c, String issID, String label)
	throws Exception {
		Asset asset = c.createAsset(issID, label);
		return asset.ID;
	}

	/**
	 * Creates an asset w/ an asset definition.
	 */
	public static String createAsset(Client c, String issID, String label, Map<String, Object> def)
	throws Exception {
		Asset asset = c.createAsset(issID, label, def);
		return asset.ID;
	}

	/**
	 * Creates an account.
	 */
	public static String createAccount(Client c, String mgrID, String label)
	throws ChainException {
		Account.CreateRequest req = new Account.CreateRequest(label);
		Account account = c.createAccount(mgrID, req);
		return account.ID;
	}

	/**
	 * Creates an address for a specified account.
	 */
	public static String createAddress(Client c, String acctID)
	throws ChainException {
		Address address = c.createAddress(acctID);
		return address.address;
	}

	/**
	 * Issues an asset to an account ID.
	 */
	public static String issue(Client c, String assetID, String acctID, int amount)
	throws ChainException {
		// build transaction
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addIssueInput(assetID, Big);
		build.addAccountOutput(assetID, acctID, BigInteger.valueOf(amount));
		Transactor.Transaction tx = c.buildTransaction(build);

		// sign transaction
		c.signTransaction(tx);

		// submit transaction
		Transactor.SubmitResponse resp = c.submitTransaction(tx);
		return resp.transactionID;
	}

	/**
	 * Issues an asset to an address.
	 */
	public static String issueToAddress(Client c, String assetID, String addr, int amount)
	throws ChainException {
		// build transaction
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addIssueInput(assetID);
		build.addAddressOutput(assetID, addr, BigInteger.valueOf(amount));
		Transactor.Transaction tx = c.buildTransaction(build);

		// sign transaction
		c.signTransaction(tx);

		// submit transaction
		Transactor.SubmitResponse resp = c.submitTransaction(tx);
		return resp.transactionID;
	}

	/**
	 * Transacts an asset between accounts by account address.
	 */
	public static String transact(Client c, String assetID, String sndrID, String rcvrAddr, int amount)
	throws ChainException {
		// build transaction
		Transactor.BuildRequest build = new Transactor.BuildRequest();
		build.addInput(assetID, sndrID, BigInteger.valueOf(amount));
		build.addAddressOutput(assetID, rcvrAddr, BigInteger.valueOf(amount));
		Transactor.Transaction tx = c.buildTransaction(build);

		// sign transaction
		c.signTransaction(tx);

		// submit transaction
		Transactor.SubmitResponse resp = c.submitTransaction(tx);
		return resp.transactionID;
	}

	/**
	 * Returns asset definition as a whitespace stripped.
	 */
	public static String getAssetDefinition(Client c, String assetID)
	throws ChainException {
		// TODO(boymanjor): replace with a non-naive implementation if asset definitions might contain whitespace
		AuditorNode.Asset check = c.getAuditorNodeAsset(assetID);
		String definition = new String(check.definition);

		// return defintion with whitespace stripped
		return definition.replaceAll("\\s+", "");
	}

	/**
	 * Searches a list of Asset.Balances on assetID. Throws an Exception if the
	 * assetID is not found.
	 */
	public static Asset.Balance getAssetBalance(List<Asset.Balance> balances, String assetID)
	throws Exception {
		for (Asset.Balance ab : balances) {
			if (ab.assetID.equals(assetID)) {
				return ab;
			}
		}
		throw new Exception(String.format("Asset %s should exist in account balances.", assetID));
	}

	/**
	 * Retries task until successful or timeout
	 */
	public static void retry(Callable<Void> task)
	throws Exception {
		// TODO(boymanjor): Update timeout to reasonable baseline after benchmarking
		long start = System.currentTimeMillis();
		long end = start + 500;

		while (System.currentTimeMillis() < end) {
			try {
				task.call();
				return;
			} catch (Exception | AssertionError e) {
				Thread.sleep(25);
			}
		}
		// final call will succeed or throw the Exception or AssertionError
		task.call();
	}

	/**
	 * Waits for blockchain updates to propagate across network.
	 */
	public static void waitForPropagation(Client c, String txID)
	throws Exception {
		retry(() -> {
			c.getAuditorNodeTransaction(txID);
			return null;
		});
	}

	/**
	 * Validates account balance.
	 */
	public static void validateAccountBalance(Client c, String acctID, Map<String, Integer> balances)
	throws Exception {
		retry(() -> {
			Asset.BalancePage abp = c.listAccountBalances(acctID);
			List<Asset.Balance> acct = abp.balances;
			// assert amount of assets held
			int actualSz = acct.size();
			int expectedSz = balances.size();
			assert actualSz == expectedSz : TestUtils.fail("# of assets", actualSz, expectedSz);

			// assert individual asset balances
			for (Asset.Balance asset : acct) {
				int actualBal = asset.confirmed.intValue();
				int expectedBal = balances.get(asset.assetID);
				assert actualBal == expectedBal : TestUtils.fail("balance", actualBal, expectedBal);
			}
			return null;
		});
	}

	/**
	 * Validates asset issuance.
	 */
	public static void validateAssetIssuance(Client c, String assetID, int amount)
	throws Exception {
		retry(() -> {
			Asset check = c.getAsset(assetID);
			int total = check.issued.total.intValue();
			int confirmed = check.issued.confirmed.intValue();
			assert total == amount : TestUtils.fail("total", total, amount);
			assert confirmed == amount : TestUtils.fail("confirmed", confirmed, amount);
			return null;
		});
	}

	/**
	 * Validates asset definition.
	 */
	public static void validateAssetDefinition(Client c, String assetID, String defCheck)
	throws Exception {
		retry(() -> {
			String definition = TestUtils.getAssetDefinition(c, assetID);
			assert definition.equals(defCheck) : TestUtils.fail("asset definition", definition, defCheck);
			return null;
		});
	}

	/**
	 * Builds assertion message.
	 */
	public static String fail(String attr, Object actual, Object expected) {
		return String.format("%s equals %s. Should equal %s.", attr, actual, expected);
	}
}
