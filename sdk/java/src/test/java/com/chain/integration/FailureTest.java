package com.chain.integration;

import com.chain.TestUtils;
import com.chain.api.*;
import com.chain.exception.APIException;
import com.chain.exception.HTTPException;
import com.chain.http.Context;
import com.chain.signing.HsmSigner;

import org.junit.Test;

import java.net.URL;

/**
 * FailureTest asserts that single-item versions
 * of batch endpoints throw exceptions on error.
 */
public class FailureTest {
  static Context context;

  @Test
  public void run() throws Exception {
    testCreateKey();
    testCreateAccount();
    testCreateAsset();
    testCreateControlProgram();
    testBuildTransaction();
    testSignTransaction();
    testSubmitTransaction();
  }

  public void testCreateKey() throws Exception {
    try {
      MockHsm.Key.create(new Context(new URL("http://wrong")));
    } catch (HTTPException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testCreateAccount() throws Exception {
    context = TestUtils.generateContext();
    try {
      new Account.Builder().create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testCreateAsset() throws Exception {
    context = TestUtils.generateContext();
    try {
      new Asset.Builder().create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testCreateControlProgram() throws Exception {
    context = TestUtils.generateContext();
    try {
      new ControlProgram.Builder().create(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testBuildTransaction() throws Exception {
    context = TestUtils.generateContext();
    try {
      new Transaction.Builder().addAction(new Transaction.Action.Issue()).build(context);
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testSignTransaction() throws Exception {
    context = TestUtils.generateContext();
    HsmSigner.addKey(MockHsm.Key.create(context), MockHsm.getSignerContext(context));
    try {
      HsmSigner.sign(new Transaction.Template());
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }

  public void testSubmitTransaction() throws Exception {
    context = TestUtils.generateContext();
    try {
      Transaction.submit(context, new Transaction.Template());
    } catch (APIException e) {
      return;
    }
    throw new Exception("expecting APIException");
  }
}
