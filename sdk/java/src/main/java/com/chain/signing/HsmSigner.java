package com.chain.signing;

import com.chain.api.MockHsm;
import com.chain.http.BatchResponse;

import com.chain.api.Transaction;
import com.chain.exception.*;
import com.chain.http.Context;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.net.URL;
import java.net.MalformedURLException;
import java.util.*;

public class HsmSigner {
  private static Map<URL, List<String>> hsmXPubs = new HashMap();

  public static void addKey(String xpub, URL hsmUrl) {
    if (!hsmXPubs.containsKey(hsmUrl)) {
      hsmXPubs.put(hsmUrl, new ArrayList<String>());
    }
    hsmXPubs.get(hsmUrl).add(xpub);
  }

  public static void addKey(String xpub, String hsmUrl) throws BadURLException {
    try {
      addKey(xpub, new URL(hsmUrl));
    } catch (MalformedURLException e) {
      throw new BadURLException(e.getMessage());
    }
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
    for (Map.Entry<URL, List<String>> entry : hsmXPubs.entrySet()) {
      Context context = new Context(entry.getKey());
      HashMap<String, Object> body = new HashMap();
      body.put("transactions", Arrays.asList(template));
      body.put("xpubs", entry.getValue());
      template =
          context.singletonBatchRequest("sign-transaction", body, Transaction.Template.class);
    }
    return template;
  }

  // TODO(boymanjor): Currently this method trusts the hsm to return a tx template
  // in the event it is unable to sign it. Moving forward we should employ a filter
  // step and only send txs to hsms that hold the proper key material to sign.
  public static BatchResponse<Transaction.Template> signBatch(List<Transaction.Template> tmpls)
      throws ChainException {
    int[] originalIndex = new int[tmpls.size()];
    for (int i = 0; i < tmpls.size(); i++) {
      originalIndex[i] = i;
    }

    Map<Integer, APIException> errors = new HashMap<>();

    for (Map.Entry<URL, List<String>> entry : hsmXPubs.entrySet()) {
      Context hsm = new Context(entry.getKey());

      HashMap<String, Object> requestBody = new HashMap();
      requestBody.put("transactions", tmpls);
      requestBody.put("xpubs", entry.getValue());

      BatchResponse<Transaction.Template> batch =
          hsm.batchRequest("sign-transaction", requestBody, Transaction.Template.class);

      // We need to work towards a single, final BatchResponse that uses the
      // original indexes. For the next cycle, we should retain only those
      // templates for which the most recent sign response was successful, and
      // maintain a mapping of each template's index in the upcoming request
      // to its original index.

      List<Transaction.Template> nextTmpls = new ArrayList<>();
      int[] nextOriginalIndex = new int[batch.successesByIndex().size()];

      for (int i = 0; i < tmpls.size(); i++) {
        if (batch.isSuccess(i)) {
          nextTmpls.add(batch.successesByIndex().get(i));
          nextOriginalIndex[nextTmpls.size() - 1] = originalIndex[i];
        } else {
          errors.put(originalIndex[i], batch.errorsByIndex().get(i));
        }
      }

      tmpls = nextTmpls;
      originalIndex = nextOriginalIndex;

      // Early out if we have no templates remaining for the next cycle.
      if (tmpls.isEmpty()) {
        break;
      }
    }

    Map<Integer, Transaction.Template> successes = new HashMap<>();
    for (int i = 0; i < tmpls.size(); i++) {
      successes.put(originalIndex[i], tmpls.get(i));
    }

    return new BatchResponse<Transaction.Template>(successes, errors);
  }
}
