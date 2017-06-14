package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.*;
import com.chain.exception.APIException;
import com.chain.exception.ConfigurationException;
import com.chain.exception.HTTPException;
import com.chain.http.Client;
import com.chain.signing.HsmSigner;

import org.junit.Test;

import java.net.URL;

/**
 * FailureTest asserts that single-item versions
 * of batch endpoints throw exceptions on error.
 */
public class FailureTest {
  static Client client;

  @Test
  public void run() throws Exception {
    testCreateKey();
    testCreateAccount();
    testCreateAsset();
    testBuildTransaction();
    testSignTransaction();
    testSubmitTransaction();
  }

  public void testCreateKey() throws Exception {
    try {
      MockHsm.Key.create(new Client(new URL("http://wrong")));
    } catch (ConfigurationException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testCreateAccount() throws Exception {
    client = TestUtils.generateClient();
    try {
      new Account.Builder().create(client);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testCreateAsset() throws Exception {
    client = TestUtils.generateClient();
    try {
      new Asset.Builder().create(client);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testBuildTransaction() throws Exception {
    client = TestUtils.generateClient();
    try {
      new Transaction.Builder().addAction(new Transaction.Action.Issue()).build(client);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testSignTransaction() throws Exception {
    client = TestUtils.generateClient();
    HsmSigner.addKey(MockHsm.Key.create(client), MockHsm.getSignerClient(client));
    try {
      HsmSigner.sign(new Transaction.Template());
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testSubmitTransaction() throws Exception {
    client = TestUtils.generateClient();
    try {
      Transaction.submit(client, new Transaction.Template());
    } catch (APIException e) {
      if (!"CH730".equals(e.code)) {
        throw new Exception("expecting CH730 code, got " + e.code);
      }
      return;
    }
    throw new Exception("expecting APIException");
  }
}
