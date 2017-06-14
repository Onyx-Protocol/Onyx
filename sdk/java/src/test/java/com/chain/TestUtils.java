package com.chain;

import com.chain.exception.BadURLException;
import com.chain.http.Client;

/**
 * TestUtils provides a simplified api for testing.
 */
public class TestUtils {
  public static Client generateClient() throws Exception {
    String coreURL = System.getProperty("chain.api.url");
    String accessToken = System.getProperty("client.access.token");
    String certPath = System.getProperty("client.trusted.cert");
    if (coreURL == null || coreURL.isEmpty()) {
      coreURL = "http://localhost:1999";
    }
    return new Client.Builder()
            .setURL(coreURL)
            .setAccessToken(accessToken)
            .setTrustedCerts(certPath)
            .build();
  }

  public static Client generateSignerClient() throws Exception {
    String coreURL = System.getProperty("chain.api.url");
    String accessToken = System.getProperty("client.access.token");
    String certPath = System.getProperty("client.trusted.cert");
    if (coreURL == null || coreURL.isEmpty()) {
      coreURL = "http://localhost:1999";
    }
    coreURL = coreURL + "/mockhsm";
    return new Client.Builder()
            .setURL(coreURL)
            .setAccessToken(accessToken)
            .setTrustedCerts(certPath)
            .build();
  }
}
