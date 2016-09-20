package com.chain;

import java.net.MalformedURLException;
import java.net.URL;
import java.util.concurrent.Callable;

/**
 * TestUtils provides a simplified api for testing.
 */
public class TestUtils {

  /**
   * Retries task until successful or timeout
   */
  public static void retry(Callable<Void> task) throws Exception {
    // TODO(boymanjor): Update timeout to reasonable baseline after benchmarking
    long start = System.currentTimeMillis();
    long end = start + 500;

    while (System.currentTimeMillis() < end) {
      try {
        task.call();
        return;
      } catch (Exception | AssertionError e) {
        Thread.sleep(25);
      }
    }
    // final call will succeed or throw the Exception (or AssertionError)
    task.call();
  }

  /**
   * Builds assertion message.
   */
  public static String fail(String attr, Object actual, Object expected) {
    return String.format("%s equals %s. Should equal %s.", attr, actual, expected);
  }

  public static URL getCoreURL(String coreURL) throws MalformedURLException {
    if (coreURL == null || coreURL.isEmpty()) {
      return new URL("http://localhost:8080");
    } else {
      return new URL(coreURL);
    }
  }
}
