package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;
import com.chain.http.Client;
import com.chain.proto.*;

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
    public byte[] xpub;

    private Key(XPub proto) {
      this.alias = proto.getAlias();
      this.xpub = proto.getXpub().toByteArray();
    }

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
      return create(client, null);
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
      CreateKeyRequest.Builder req = CreateKeyRequest.newBuilder();
      if (alias != null) {
        req.setAlias(alias);
      }
      CreateKeyResponse resp = client.hsm().createKey(req.build());
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      return new Key(resp.getXpub());
    }

    /**
     * A paged collection of key objects returned from the core.
     */
    public static class Items extends PagedItems<Key, ListKeysQuery> {
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
        ListKeysResponse resp = this.client.hsm().listKeys(this.next);
        if (resp.hasError()) {
          throw new APIException(resp.getError());
        }

        Items items = new Items();
        for (com.chain.proto.XPub key : resp.getItemsList()) {
          items.list.add(new Key(key));
        }
        items.lastPage = resp.getLastPage();
        items.next = resp.getNext();
        items.setClient(this.client);
        return items;
      }

      public void setNext(Query query) {
        ListKeysQuery.Builder builder = ListKeysQuery.newBuilder();
        if (query.aliases != null && !query.aliases.isEmpty()) {
          builder.addAllAliases(query.aliases);
        }
        if (query.after != null && !query.after.isEmpty()) {
          builder.setAfter(query.after);
        }

        this.next = builder.build();
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
