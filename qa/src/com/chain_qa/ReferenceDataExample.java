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
import java.util.HashMap;
import java.util.Map;

public class ReferenceDataExample {
    public static void main(String[] args) throws Exception {
        System.out.print("Running...");
        Context context = new Context(TestUtils.getCoreURL(System.getenv("CHAIN_API_URL")));
        MockHsm.Key mainKey = MockHsm.Key.create(context);
        HsmSigner.addKey(mainKey);

        new Account.Builder()
                .setAlias("alice")
                .addXpub(mainKey.xpub)
                .setQuorum(1)
                .addTag("type", "checking")
                .addTag("first_name", "Alice")
                .addTag("last_name", "Jones")
                .addTag("user_id", "12345")
                .addTag("status", "enabled")
                .create(context);

        new Account.Builder()
                .setAlias("bob")
                .addXpub(mainKey.xpub)
                .setQuorum(1)
                .addTag("type", "checking")
                .addTag("first_name", "Bob")
                .addTag("last_name", "Smith")
                .addTag("user_id", "67890")
                .addTag("status", "enabled")
                .create(context);

        Map<String, Object> def = new HashMap<>();
        def.put("type", "security");
        def.put("sub-type", "corporate-bond");
        def.put("entity", "Acme Inc.");
        def.put("maturity", "2016-09-01T18:24:47+00:00");
        new Asset.Builder()
                .setAlias("acme-bond")
                .addXpub(mainKey.xpub)
                .setQuorum(1)
                .addTag("internal_rating", "B")
                .setDefinition(def)
                .create(context);

        Map<String, Object> txref = new HashMap<>();
        txref.put("external_reference", "12345");
        Transaction.Template tx1 = new Transaction.Builder()
                .issueByAlias("acme-bond", BigInteger.valueOf(100), null)
                .controlWithAccountByAlias("alice", "acme-bond", BigInteger.valueOf(100), null)
                .setReferenceData(txref)
                .build(context);
        Transaction.submit(context, HsmSigner.sign(Arrays.asList(tx1)));

        Map<String, Object> retref = new HashMap<>();
        retref.put("external_reference", "67890");
        Transaction.Template tx2 = new Transaction.Builder()
                .spendFromAccountByAlias("alice", "acme-bond", BigInteger.valueOf(100), null)
                .retireByAlias("acme-bond", BigInteger.valueOf(100), retref)
                .build(context);
        Transaction.submit(context, HsmSigner.sign(Arrays.asList(tx2)));
        System.out.print("done");
    }
}