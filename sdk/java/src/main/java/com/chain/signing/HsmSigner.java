package com.chain.signing;

import com.chain.api.MockHsm;

import com.chain.api.Transaction;
import com.chain.exception.APIException;
import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.net.URL;
import java.util.*;

public class HsmSigner {
  private static Map<URL, List<String>> hsmXPubs = new HashMap();

  public static void addKey(String xpub, URL hsmUrl) {
    if (!hsmXPubs.containsKey(hsmUrl)) {
      hsmXPubs.put(hsmUrl, new ArrayList<String>());
    }
    hsmXPubs.get(hsmUrl).add(xpub);
  }

  public static void addKey(MockHsm.Key key) {
    addKey(key.xpub, key.hsmUrl);
  }

  public static void addKeys(List<MockHsm.Key> keys) {
    for (MockHsm.Key key : keys) {
      addKey(key.xpub, key.hsmUrl);
    }
  }

  public static Transaction.Template sign(Transaction.Template template) throws ChainException {
    List<Transaction.Template> templates = signBatch(Arrays.asList(template));
    Transaction.Template response = templates.get(0);
    if (response.code != null) {
      throw new APIException(template.code, template.message, template.detail, null);
    }
    return response;
  }

  // TODO(boymanjor): Currently this method trusts the hsm to return a tx template
  // in the event it is unable to sign it. Moving forward we should employ a filter
  // step and only send txs to hsms that hold the proper key material to sign.
  public static List<Transaction.Template> signBatch(List<Transaction.Template> tmpls)
      throws ChainException {
    for (Map.Entry<URL, List<String>> entry : hsmXPubs.entrySet()) {
      Context hsm = new Context(entry.getKey());
      Type type = new TypeToken<ArrayList<Transaction.Template>>() {}.getType();

      HashMap<String, Object> requestBody = new HashMap();
      requestBody.put("transactions", tmpls);
      requestBody.put("xpubs", entry.getValue());

      tmpls = hsm.request("sign-transaction", requestBody, type);
    }
    return tmpls;
  }
}
