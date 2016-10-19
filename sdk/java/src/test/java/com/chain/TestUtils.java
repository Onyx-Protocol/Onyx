package com.chain;

import com.chain.http.Client;

import java.net.MalformedURLException;
import java.net.URL;

/**
 * TestUtils provides a simplified api for testing.
 */
public class TestUtils {
  public static Client generateClient() throws MalformedURLException {
    String coreURL = System.getProperty("chain.api.url");
    String accessToken = System.getProperty("client.access.token");
    if (coreURL == null || coreURL.isEmpty()) {
      coreURL = "http://localhost:1999";
    }
    if (accessToken == null || accessToken.isEmpty()) {
      return new Client(new URL(coreURL));
    }
    return new Client(new URL(coreURL), accessToken);
  }
}
