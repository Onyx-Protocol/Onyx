package com.chain.signing;

import com.chain.api.MockHsm;
import com.chain.api.TransactionTemplate;
import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.net.URL;
import java.util.*;

public class HsmSigner {
    private static Set<URL> hsmUrls = new HashSet<>();

    public static void addKey(String xpub, URL hsmUrl) {
        hsmUrls.add(hsmUrl);
    }

    public static void addKey(MockHsm.Key key) {
        hsmUrls.add(key.hsmUrl);
    }

    public static void addKeys(List<MockHsm.Key> keys) {
        for (MockHsm.Key key : keys) {
            hsmUrls.add(key.hsmUrl);
        }
    }

    // TODO(boymanjor): Currently this method trusts the hsm to return a tx template
    // in the event it is unable to sign it. Moving forward we should employ a filter
    // step and only send txs to hsms that hold the proper key material to sign.
    public static List<TransactionTemplate> sign(List<TransactionTemplate> tmpls)
    throws ChainException {
        for (URL hsmUrl : hsmUrls) {
            Context hsm = new Context(hsmUrl);
            Type type = new TypeToken<ArrayList<TransactionTemplate>>() {}.getType();
            tmpls = hsm.request("sign-transaction-template", tmpls, type);
        }
        return tmpls;
    }
}
