package com.chain;

import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.http.Client;

/**
 * TestUtils provides a simplified api for testing.
 */
public class TestUtils {
  public static Client generateClient() throws ChainException {
    String coreURL = System.getProperty("chain.api.url");
    String accessToken = System.getProperty("client.access.token");
    if (coreURL == null || coreURL.isEmpty()) {
      coreURL = "http://localhost:1999";
    }
    return new Client(coreURL, accessToken);
  }
}
