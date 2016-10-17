package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;
import com.chain.http.*;
import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.util.*;

/**
 * A single Account on the Chain Core, capable of spending or receiving assets in a transaction.
 */
public class Account {
  /**
   * Unique account identifier.
   */
  public String id;

  /**
   * User specified, unique identifier.
   */
  public String alias;

  /**
   * The list of keys used to create control programs under the account.<br>
   * Signatures from these keys are required for spending funds held in the account.
   */
  public Key[] keys;

  /**
   * The number of keys required to sign transactions for the account.
   */
  public int quorum;

  /**
   * User-specified tag structure for the account.
   */
  public Map<String, Object> tags;

  /**
   * A class storing information about the keys associated with the account.
   */
  public static class Key {
    /**
     * Hex-encoded representation of the root extended public key
     */
    @SerializedName("root_xpub")
    public String rootXpub;

    /**
     * The extended public key used to create control programs for the account.
     */
    @SerializedName("account_xpub")
    public String accountXpub;

    /**
     * The derivation path of the extended key.
     */
    @SerializedName("account_derivation_path")
    public String[] derivationPath;
  }

  /**
   * A paged collection of accounts returned from a query.
   */
  public static class Items extends PagedItems<Account> {
    /**
     * Requests a page of accounts based on an underlying query.
     * @return a page of accounts objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the accounts.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items getPage() throws ChainException {
      Items items = this.context.request("list-accounts", this.next, Items.class);
      items.setContext(this.context);
      return items;
    }
  }

  /**
   * A builder class for generating account queries.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Executes a query on the core's accounts.
     * @param ctx context object that makes requests to the core
     * @return a collection of account objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the accounts.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items execute(Context ctx) throws ChainException {
      Items items = new Items();
      items.setContext(ctx);
      items.setNext(this.next);
      return items.getPage();
    }
  }

  /**
   * A builder class for creating account objects.
   */
  public static class Builder {
    /**
     * User specified, unique identifier.
     */
    public String alias;

    /**
     * The number of keys required to sign transactions for the account.
     */
    public int quorum;

    /**
     * The list of keys used to create control programs under the account.<br>
     * Signatures from these keys are required for spending funds held in the account.
     */
    @SerializedName("root_xpubs")
    public List<String> rootXpubs;

    /**
     * User-specified tag structure for the account.
     */
    public Map<String, Object> tags;

    /**
     * Unique identifier used for request idempotence.
     */
    @SerializedName("client_token")
    private String clientToken;

    /**
     * Default constructor initializes the list of keys.
     */
    public Builder() {
      this.rootXpubs = new ArrayList<>();
    }

    /**
     * Creates an account object.
     * @param ctx context object that makes request to the core
     * @return an account object
     * @throws APIException This exception is raised if the api returns errors while creating the account.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Account create(Context ctx) throws ChainException {
      return ctx.singletonBatchRequest("create-account", Arrays.asList(this), Account.class);
    }

    /**
     * Creates a batch of account objects.
     * <strong>Note:</strong> this method will not throw an exception APIException. Each builder's response object must be checked for error.
     * @param ctx context object that makes requests to the core
     * @param builders list of account builders
     * @return a list of account and/or error objects
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public static BatchResponse<Account> createBatch(Context ctx, List<Builder> builders)
        throws ChainException {
      for (Builder builder : builders) {
        builder.clientToken = UUID.randomUUID().toString();
      }
      return ctx.batchRequest("create-account", builders, Account.class);
    }

    /**
     * Sets the alias on the builder object.
     * @param alias alias
     * @return updated builder object
     */
    public Builder setAlias(String alias) {
      this.alias = alias;
      return this;
    }

    /**
     * Sets the quorum for control programs.
     * @param quorum proposed quorum
     * @return updated builder object
     */
    public Builder setQuorum(int quorum) {
      this.quorum = quorum;
      return this;
    }

    /**
     * Adds a key to the builder's list.
     * @param xpub key
     * @return updated builder object.
     */
    public Builder addRootXpub(String xpub) {
      this.rootXpubs.add(xpub);
      return this;
    }

    /**
     * Sets the builder's list of keys.
     * <strong>Note:</strong> any existing keys will be replaced.
     * @param xpubs list of xpubs
     * @return updated builder object
     */
    public Builder setRootXpubs(List<String> xpubs) {
      this.rootXpubs = new ArrayList<>();
      for (String xpub : xpubs) {
        this.rootXpubs.add(xpub);
      }
      return this;
    }

    /**
     * Adds a field to the existing account tags object (initializing the object if it doesn't exist).
     * @param key key of the tag
     * @param value value of the tag
     * @return updated builder object
     */
    public Builder addTag(String key, Object value) {
      if (this.tags == null) {
        this.tags = new HashMap<>();
      }
      this.tags.put(key, value);
      return this;
    }

    /**
     * Sets the account tags object.<br>
     * <strong>Note:</strong> any existing account tag fields will be replaced.
     * @param tags account tags object
     * @return updated builder object
     */
    public Builder setTags(Map<String, Object> tags) {
      this.tags = tags;
      return this;
    }
  }
}
