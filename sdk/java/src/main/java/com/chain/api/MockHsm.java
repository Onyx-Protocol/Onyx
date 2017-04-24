package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;
import com.chain.http.Client;

import java.net.MalformedURLException;
import java.net.URL;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * A mock HSM provided by Chain Core to handle key material in development.
 */
public class MockHsm {
  /**
   * Returns a new client that knows how to make requests to the mock HSM.
   * @param client client object that makes request to the core
   * @return new client object
   * @throws BadURLException
   */
  public static Client getSignerClient(Client client) throws ChainException {
    try {
      List<URL> urls = new ArrayList<>();
      for (URL url : client.urls()) {
        urls.add(new URL(url.toString() + "/mockhsm"));
      }
      return new Client.Builder(client).setURLs(urls).build();
    } catch (MalformedURLException e) {
      throw new BadURLException(e.getMessage());
    }
  }

  /**
   * A class representing an extended public key. An instance of this class
   * stores a link to the mock HSM holding the corresponding private key.
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
     * Creates a key object.
     * @param client client object that makes requests to the core
     * @return a key object
     * @throws APIException This exception is raised if the api returns errors while creating the key.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public static Key create(Client client) throws ChainException {
      Key key = client.request("mockhsm/create-key", null, Key.class);
      return key;
    }

    /**
     * Creates a key object.
     * @param client client object that makes requests to the core
     * @param alias user specified identifier
     * @return a key object
     * @throws APIException This exception is raised if the api returns errors while creating the key.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public static Key create(Client client, String alias) throws ChainException {
      Map<String, Object> req = new HashMap<>();
      req.put("alias", alias);
      Key key = client.request("mockhsm/create-key", req, Key.class);
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
        Items items = this.client.request("mockhsm/list-keys", this.next, Items.class);
        items.setClient(this.client);
        return items;
      }
    }

    public static class QueryBuilder {
      private Query query;

      public QueryBuilder() {
        query = new Query();
      }

      public QueryBuilder setAliases(List<String> aliases) {
        query.aliases = new ArrayList<>(aliases);
        return this;
      }

      public QueryBuilder addAlias(String alias) {
        query.aliases.add(alias);
        return this;
      }

      /**
       * Retrieves a page of key objects from the core.
       * @param client client object that makes requests to the core
       * @return a collection of key objects
       * @throws APIException This exception is raised if the api returns errors while retrieving the keys.
       * @throws BadURLException This exception wraps java.net.MalformedURLException.
       * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
       * @throws HTTPException This exception is raised when errors occur making http requests.
       * @throws JSONException This exception is raised due to malformed json requests or responses.
       */
      public Items execute(Client client) throws ChainException {
        Items items = new Items();
        items.setClient(client);
        items.setNext(query);
        return items.getPage();
      }
    }
  }
}
