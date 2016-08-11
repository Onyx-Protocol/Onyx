package com.chain.signing;

import com.chain.api.TransactionTemplate;
import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.net.URL;
import java.util.*;

public class HsmSigner {
    private List<URL> hsmUrls;

    public HsmSigner() {
        this.hsmUrls = new ArrayList<>();
    }

    public void addKey(String xpub, URL hsmUrl) {
        if (!hsmUrls.contains(hsmUrl)) {
            hsmUrls.add(hsmUrl);
        }
    }

    // TODO(boymanjor): Currently this method trusts the hsm to return a tx template
    // in the event it is unable to sign it. Moving forward we should employ a filter
    // step and only send txs to hsms that hold the proper key material to sign.
    public List<TransactionTemplate> sign(List<TransactionTemplate> tmpls)
    throws ChainException {
        for (URL hsmUrl : hsmUrls) {
            Context hsm = new Context(hsmUrl);
            Type type = new TypeToken<ArrayList<TransactionTemplate>>() {}.getType();
            tmpls = hsm.request("sign-transaction-template", tmpls, type);
        }
        return tmpls;
    }
}
