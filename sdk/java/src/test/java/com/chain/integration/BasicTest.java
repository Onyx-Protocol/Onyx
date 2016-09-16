package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.exception.APIException;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;
import org.junit.Test;

import java.math.BigInteger;
import java.util.Arrays;
import java.util.List;

public class BasicTest {
	final String ALICE = "basic-alice";
	final String BOB = "basic-bob";
	final String ASSET = "basic-asset";
	
	@Test
	public void test() throws Exception {
		Context context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
		MockHsm.Key mainKey = MockHsm.Key.create(context);
		HsmSigner.addKey(mainKey);

		new Account.Builder()
				.setAlias(ALICE)
				.addXpub(mainKey.xpub)
				.setQuorum(1)
				.create(context);

		new Account.Builder()
				.setAlias(BOB)
				.addXpub(mainKey.xpub)
				.setQuorum(1)
				.create(context);

		new Asset.Builder()
				.setAlias(ASSET)
				.addXpub(mainKey.xpub)
				.setQuorum(1)
				.create(context);

		Transaction.Template issuance = new Transaction.Builder()
				.issueByAlias(ASSET, BigInteger.valueOf(100), null)
				.controlWithAccountByAlias(ALICE, ASSET, BigInteger.valueOf(100), null)
				.build(context);
		List<Transaction.SubmitResponse> responses = Transaction.submit(context, HsmSigner.sign(Arrays.asList(issuance)));
		for (Transaction.SubmitResponse resp : responses) {
			if (resp.id == null) {
				throw new APIException(resp.code, resp.message, resp.detail, null);
			}
		}

		Transaction.Template spending = new Transaction.Builder()
				.spendFromAccountByAlias(ALICE, ASSET, BigInteger.valueOf(10), null)
				.controlWithAccountByAlias(BOB, ASSET, BigInteger.valueOf(10), null)
				.build(context);
		responses = Transaction.submit(context, HsmSigner.sign(Arrays.asList(spending)));
		for (Transaction.SubmitResponse resp : responses) {
			if (resp.id == null) {
				throw new APIException(resp.code, resp.message, resp.detail, null);
			}
		}

		Transaction.Template retirement = new Transaction.Builder()
				.spendFromAccountByAlias(BOB, ASSET, BigInteger.valueOf(5), null)
				.retireByAlias(ASSET, BigInteger.valueOf(5), null)
				.build(context);
		responses = Transaction.submit(context, HsmSigner.sign(Arrays.asList(retirement)));
		for (Transaction.SubmitResponse resp : responses) {
			if (resp.id == null) {
				throw new APIException(resp.code, resp.message, resp.detail, null);
			}
		}
	}
}