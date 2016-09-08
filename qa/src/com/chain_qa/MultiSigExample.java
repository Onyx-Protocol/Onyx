package com.chain_qa;

import com.chain.api.Account;
import com.chain.api.Asset;
import com.chain.api.MockHsm;
import com.chain.api.Transaction;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;

import java.math.BigInteger;
import java.net.URL;
import java.util.Arrays;

public class MultiSigExample {
    public static void main(String[] args) throws Exception {
        System.out.print("Running...");
        Context context = new Context(TestUtils.getCoreURL(System.getenv("CHAIN_API_URL")));
        MockHsm.Key key1 = MockHsm.Key.create(context);
        MockHsm.Key key2 = MockHsm.Key.create(context);
        MockHsm.Key key3 = MockHsm.Key.create(context);
        HsmSigner.addKeys(Arrays.asList(key1, key2, key3));

        new Account.Builder()
                .setAlias("alice")
                .addXpub(key1.xpub)
                .addXpub(key2.xpub)
                .addXpub(key3.xpub)
                .setQuorum(2)
                .create(context);

        new Asset.Builder()
                .setAlias("gold")
                .addXpub(key1.xpub)
                .addXpub(key2.xpub)
                .addXpub(key3.xpub)
                .setQuorum(2)
                .create(context);

        Transaction.Template tx = new Transaction.Builder()
                .issueByAlias("gold", BigInteger.valueOf(100), null)
                .controlWithAccountByAlias("alice", "gold", BigInteger.valueOf(100), null)
                .build(context);
        Transaction.submit(context, HsmSigner.sign(Arrays.asList(tx)));
        System.out.print("done");
    }
}
