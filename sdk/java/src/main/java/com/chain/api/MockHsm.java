package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;
import com.chain.http.Context;

import java.net.MalformedURLException;
import java.net.URL;
import java.util.HashMap;
import java.util.Map;

/**
 * A mock hsm provided by Chain Core to handle key material in development.
 */
public class MockHsm {
  /**
   * A class representing an extended public key.<br>
   * An instance of this class stores a link to the mock hsm holding the corresponding private key.
   */
  public static class Key {
    /**
     * User specified, unique identifier of the key.
     */
    public String alias;

    /**
     * Hex-encoded string representation of the key.
     */
    public String xpub;

    /**
     * The URL of the mock hsm which stores the key.
     */
    public URL hsmUrl;

    /**
     * Creates a key object.
     * @param ctx context object that makes requests to the core
     * @return a key object
     * @throws APIException This exception is raised if the api returns errors while creating the key.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public static Key create(Context ctx) throws ChainException {
      Key key = ctx.request("mockhsm/create-key", null, Key.class);
      key.hsmUrl = buildMockHsmUrl(ctx.getUrl());
      return key;
    }

    /**
     * Creates a key object.
     * @param ctx context object that makes requests to the core
     * @param alias user specified identifier
     * @return a key object
     * @throws APIException This exception is raised if the api returns errors while creating the key.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public static Key create(Context ctx, String alias) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("alias", alias);
      Key key = ctx.request("mockhsm/create-key", req, Key.class);
      key.hsmUrl = buildMockHsmUrl(ctx.getUrl());
      return key;
    }

    /**
     * A paged collection of key objects returned from the core.
     */
    public static class Items extends PagedItems<Key> {
      /**
       * Requests a page of key objects from the core.
       * @return a collection of key objects
       * @throws APIException This exception is raised if the api returns errors while retrieving the keys.
       * @throws BadURLException This exception wraps java.net.MalformedURLException.
       * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
       * @throws HTTPException This exception is raised when errors occur making http requests.
       * @throws JSONException This exception is raised due to malformed json requests or responses.
       */
      @Override
      public Items getPage() throws ChainException {
        Items items = this.context.request("mockhsm/list-keys", this.next, Items.class);
        items.setContext(this.context);
        URL mockHsmUrl = buildMockHsmUrl(this.context.getUrl());
        for (Key k : items.list) {
          k.hsmUrl = mockHsmUrl;
        }
        return items;
      }
    }

    /**
     * Retrieves a page of key objects from the core.
     * @param ctx context object that makes requests to the core
     * @return a collection of key objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the keys.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public static Items list(Context ctx) throws ChainException {
      Items items = new Items();
      items.setContext(ctx);
      return items.getPage();
    }
  }

  /**
   * Creates a mock hsm url object given a Chain Core url.
   * @param coreUrl the Chain Core url
   * @return a mock hsm url object
   * @throws BadURLException thrown if a MalformedURLException is thrown while building the URL object
   */
  private static URL buildMockHsmUrl(URL coreUrl) throws BadURLException {
    try {
      return new URL(coreUrl.toString() + "/mockhsm");
    } catch (MalformedURLException e) {
      throw new BadURLException(e.getMessage());
    }
  }
}
