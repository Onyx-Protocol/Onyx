package com.chain.http;

import com.chain.exception.*;

import java.lang.reflect.Type;
import java.net.*;
import java.util.concurrent.TimeUnit;

/**
 * The Context object contains all information necessary to
 * perform an HTTP request against a remote API.
 */
public class Context {

  private URL url;
  private String accessToken;
  private APIClient httpClient;

  /**
   * Create a new http Context object using the default development host URL.
   */
  public Context() {
    URL url;
    try {
      url = new URL("http://localhost:1999");
    } catch (Exception e) {
      throw new RuntimeException("invalid default development URL");
    }

    this.url = url;
    this.httpClient = new APIClient(url);
  }

  /**
   * Create a new http Context object
   *
   * @param url The URL of the Chain Core or HSM
   */
  public Context(URL url) {
    this.url = url;
    this.httpClient = new APIClient(url);
  }

  /**
   * Create a new http Context object
   *
   * @param url The URL of the Chain Core or HSM
   * @param accessToken A Client API access token.
   */
  public Context(URL url, String accessToken) {
    this.url = url;
    this.accessToken = accessToken;
    this.httpClient = new APIClient(url, accessToken);
  }

  /**
   * Perform a single HTTP POST request against the API for a specific action.
   *
   * @param action The requested API action
   * @param body Body payload sent to the API as JSON
   * @param tClass Type of object to be deserialized from the repsonse JSON
   * @return the result of the post request
   * @throws ChainException
   */
  public <T> T request(String action, Object body, Type tClass) throws ChainException {
    return httpClient.post(action, body, tClass);
  }

  /**
   * Perform a single HTTP POST request against the API for a specific action.
   * Use this method if you want single-item semantics (creating single assets,
   * building single transactions) but the API endpoint is implemented as a
   * batch call.
   *
   * @param action The requested API action
   * @param body Body payload sent to the API as JSON
   * @param tClass Type of object to be deserialized from the repsonse JSON
   * @return the result of the post request
   * @throws ChainException
   */
  public <T> T singletonBatchRequest(String action, Object body, Type tClass)
      throws ChainException {
    return httpClient.post(action, body, tClass, true);
  }

  public URL getUrlWithBasicAuth() throws BadURLException {
    if (this.accessToken == null || this.accessToken.isEmpty()) {
      return this.url;
    }
    try {
      return new URL(String.format("%s://%s@%s", this.url.getProtocol(), this.accessToken, this.url.getAuthority()));
    } catch (MalformedURLException e) {
      throw new BadURLException(e.getMessage());
    }
  }

  public void setConnectTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setConnectTimeout(timeout, unit);
  }

  public void setReadTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setReadTimeout(timeout, unit);
  }

  public void setWriteTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setWriteTimeout(timeout, unit);
  }
}
