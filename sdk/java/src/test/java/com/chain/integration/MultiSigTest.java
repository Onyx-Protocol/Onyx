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

public class MultiSigTest {
  final String ALICE = "multisig-alice";
  final String ASSET = "multisig-asset";

  @Test
  public void test() throws Exception {
    Context context = new Context(TestUtils.getCoreURL(System.getProperty("chain.api.url")));
    MockHsm.Key key1 = MockHsm.Key.create(context);
    MockHsm.Key key2 = MockHsm.Key.create(context);
    MockHsm.Key key3 = MockHsm.Key.create(context);
    HsmSigner.addKeys(Arrays.asList(key1, key2, key3));

    new Account.Builder()
        .setAlias(ALICE)
        .addRootXpub(key1.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(2)
        .create(context);

    new Asset.Builder()
        .setAlias(ASSET)
        .addRootXpub(key1.xpub)
        .addRootXpub(key2.xpub)
        .addRootXpub(key3.xpub)
        .setQuorum(2)
        .create(context);

    Transaction.Template tx =
        new Transaction.Builder()
            .issueByAlias(ASSET, BigInteger.valueOf(100), null)
            .controlWithAccountByAlias(ALICE, ASSET, BigInteger.valueOf(100), null)
            .build(context);
    List<Transaction.SubmitResponse> responses =
        Transaction.submit(context, HsmSigner.sign(Arrays.asList(tx)));
    for (Transaction.SubmitResponse resp : responses) {
      if (resp.id == null) {
        throw new APIException(resp.code, resp.message, resp.detail, null);
      }
    }
  }
}
