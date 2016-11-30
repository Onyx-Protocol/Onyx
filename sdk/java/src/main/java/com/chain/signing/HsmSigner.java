package com.chain.signing;

import com.chain.api.MockHsm;
import com.chain.api.Util;
import com.chain.http.BatchResponse;

import com.chain.api.Transaction;
import com.chain.exception.*;
import com.chain.http.Client;
import com.chain.proto.SignTxsRequest;
import com.chain.proto.TxTemplate;
import com.chain.proto.TxsResponse;
import com.google.protobuf.ByteString;

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
  private static Map<Client, List<ByteString>> hsmXPubs = new HashMap();

  /**
   * Adds an entry to the HsmSigner's hsm client-to-keys map.
   * @param xpub the public key
   * @param hsm the hsm object
   */
  public static void addKey(String xpub, Client hsm) {
    addKey(ByteString.copyFrom(Util.hexStringToByteArray(xpub)), hsm);
  }

  public static void addKey(byte[] xpub, Client hsm) {
    addKey(ByteString.copyFrom(xpub), hsm);
  }

  private static void addKey(ByteString xpub, Client hsm) {
    if (!hsmXPubs.containsKey(hsm)) {
      hsmXPubs.put(hsm, new ArrayList<ByteString>());
    }
    hsmXPubs.get(hsm).add(xpub);
  }

  /**
   * Adds an entry to the HsmSigner's HSM client-to-keys map.
   * @param key the mockhsm key
   * @param hsm the hsm object
   */
  public static void addKey(MockHsm.Key key, Client hsm) {
    addKey(ByteString.copyFrom(key.xpub), hsm);
  }

  /**
   * Adds an entry to the HsmSigner's HSM client-to-keys map.
   * @param keys the list of mockhsm keys
   * @param hsm the hsm object
   */
  public static void addKeys(List<MockHsm.Key> keys, Client hsm) {
    for (MockHsm.Key key : keys) {
      addKey(key, hsm);
    }
  }

  /**
   * Sends a transaction template to a remote HSM for signing.
   * @param template transaction template to be signed
   * @return a signed transaction template
   * @throws ChainException
   */
  public static Transaction.Template sign(Transaction.Template template) throws ChainException {
    BatchResponse<Transaction.Template> resp = signBatch(Arrays.asList(template));
    if (resp.isError(0)) {
      throw resp.errorsByIndex().get(0);
    }
    return resp.successesByIndex().get(0);
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
    List<Integer> originalIndex = new ArrayList();
    for (int i = 0; i < tmpls.size(); i++) {
      originalIndex.add(i);
    }

    Map<Integer, APIException> errors = new HashMap<>();

    List<TxTemplate> protos = new ArrayList();
    for (Transaction.Template tmpl : tmpls) {
      protos.add(tmpl.toProtobuf());
    }

    for (Map.Entry<Client, List<ByteString>> entry : hsmXPubs.entrySet()) {
      Client hsm = entry.getKey();
      SignTxsRequest.Builder req = SignTxsRequest.newBuilder();
      req.addAllXpubs(entry.getValue());
      req.addAllTransactions(protos);

      TxsResponse resp = hsm.hsm().signTxs(req.build());
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      // We need to work towards a single, final BatchResponse that uses the
      // original indexes. For the next cycle, we should retain only those
      // templates for which the most recent sign response was successful, and
      // maintain a mapping of each template's index in the upcoming request
      // to its original index.
      List<TxTemplate> nextProtos = new ArrayList<>();
      List<Integer> nextOriginalIndex = new ArrayList();

      for (int i = 0; i < resp.getResponsesCount(); i++) {
        if (resp.getResponses(i).hasError()) {
          errors.put(originalIndex.get(i), new APIException(resp.getResponses(i).getError()));
        } else {
          nextProtos.add(resp.getResponses(i).getTemplate());
          nextOriginalIndex.add(originalIndex.get(i));
        }
      }

      protos = nextProtos;
      originalIndex = nextOriginalIndex;

      // Early out if we have no templates remaining for the next cycle.
      if (protos.isEmpty()) {
        break;
      }
    }

    Map<Integer, Transaction.Template> successes = new HashMap<>();
    for (int i = 0; i < protos.size(); i++) {
      successes.put(originalIndex.get(i), new Transaction.Template(protos.get(i)));
    }

    return new BatchResponse<>(successes, errors);
  }
}
