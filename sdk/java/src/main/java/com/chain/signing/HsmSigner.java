package com.chain.signing;

import com.chain.api.MockHsm;
import com.chain.http.BatchResponse;

import com.chain.api.Transaction;
import com.chain.exception.*;
import com.chain.http.Client;

import java.util.*;

/**
 * HsmSigner makes signing requests to remote HSMs. It stores a map of client objects
 * to public keys, and routes tx template signing requests to the relevant HSM servers.
 * Only templates with keys added to the HsmSigner's map will be signed.
 */
public class HsmSigner {
  /**
   * A map of hsm objects to public keys. The list of public keys have
   * corresponding private keys stored in remote HSM servers. The hsm
   * objects are configured to make requests to the HSMs.
   */
  private static Map<Client, List<String>> hsmXPubs = new HashMap();

  /**
   * Adds an entry to the HsmSigner's hsm client-to-keys map.
   * @param xpub the public key
   * @param hsm the hsm object
   */
  public static void addKey(String xpub, Client hsm) {
    if (!hsmXPubs.containsKey(hsm)) {
      hsmXPubs.put(hsm, new ArrayList<String>());
    }
    hsmXPubs.get(hsm).add(xpub);
  }

  /**
   * Adds an entry to the HsmSigner's HSM client-to-keys map.
   * @param key the mockhsm key
   * @param hsm the hsm object
   */
  public static void addKey(MockHsm.Key key, Client hsm) {
    addKey(key.xpub, hsm);
  }

  /**
   * Adds an entry to the HsmSigner's HSM client-to-keys map.
   * @param keys the list of mockhsm keys
   * @param hsm the hsm object
   */
  public static void addKeys(List<MockHsm.Key> keys, Client hsm) {
    for (MockHsm.Key key : keys) {
      addKey(key.xpub, hsm);
    }
  }

  /**
   * Sends a transaction template to a remote HSM for signing.
   * @param template transaction template to be signed
   * @return a signed transaction template
   * @throws ChainException
   */
  public static Transaction.Template sign(Transaction.Template template) throws ChainException {
    for (Map.Entry<Client, List<String>> entry : hsmXPubs.entrySet()) {
      Client hsm = entry.getKey();
      HashMap<String, Object> body = new HashMap();
      body.put("transactions", Arrays.asList(template));
      body.put("xpubs", entry.getValue());
      template =
          hsm.singletonBatchRequest(
              "sign-transaction", body, Transaction.Template.class, APIException.class);
    }
    return template;
  }

  /**
   * Sends a batch of transaction templates to remote HSMs for signing.
   * @param tmpls transaction templates to be signed
   * @return a batch of signed transaction templates
   * @throws ChainException
   */
  // TODO(boymanjor): Currently this method trusts the hsm to return a tx template
  // in the event it is unable to sign it. Moving forward we should employ a filter
  // step and only send txs to the HSM that holds the proper key material to sign.
  public static BatchResponse<Transaction.Template> signBatch(List<Transaction.Template> tmpls)
      throws ChainException {
    int[] originalIndex = new int[tmpls.size()];
    for (int i = 0; i < tmpls.size(); i++) {
      originalIndex[i] = i;
    }

    Map<Integer, APIException> errors = new HashMap<>();

    for (Map.Entry<Client, List<String>> entry : hsmXPubs.entrySet()) {
      Client hsm = entry.getKey();
      HashMap<String, Object> requestBody = new HashMap();
      requestBody.put("transactions", tmpls);
      requestBody.put("xpubs", entry.getValue());
      BatchResponse<Transaction.Template> batch =
          hsm.batchRequest(
              "sign-transaction", requestBody, Transaction.Template.class, APIException.class);

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

    return new BatchResponse<>(successes, errors);
  }
}
