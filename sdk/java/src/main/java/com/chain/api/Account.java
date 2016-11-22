package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;
import com.chain.http.*;
import com.google.gson.annotations.SerializedName;

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
    public String[] accountDerivationPath;
  }

  /**
   * Creates a batch of account objects.
   * <strong>Note:</strong> this method will not throw an exception APIException. Each builder's response object must be checked for error.
   * @param client client object that makes requests to the core
   * @param builders list of account builders
   * @return a list of account and/or error objects
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<Account> createBatch(Client client, List<Builder> builders)
      throws ChainException {
    for (Builder builder : builders) {
      builder.clientToken = UUID.randomUUID().toString();
    }
    return client.batchRequest("create-account", builders, Account.class, APIException.class);
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
      Items items = this.client.request("list-accounts", this.next, Items.class);
      items.setClient(this.client);
      return items;
    }
  }

  /**
   * Account.QueryBuilder utilizes the builder pattern to create {@link Account} queries.<br>
   * The possible parameters for each query can be found on the {@link BaseQueryBuilder} class.<br>
   * All parameters are optional, and should be set to filter the results accordingly.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Executes a query on the core's accounts.
     * @param client client object that makes requests to the core
     * @return a collection of account objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the accounts.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items execute(Client client) throws ChainException {
      Items items = new Items();
      items.setClient(client);
      items.setNext(this.next);
      return items.getPage();
    }
  }

  /**
   * Account.Builder utilizes the builder pattern to create {@link Account} objects.
   * The following attributes are required to be set: {@link #rootXpubs}, {@link #quorum}.
   */
  public static class Builder {
    /**
     * User specified, unique identifier.
     */
    public String alias;

    /**
     * The number of keys required to sign transactions for the account.<br>
     * <strong>Must set with {@link #setQuorum(int)} before calling {@link #create(Client)}.</strong>
     */
    public int quorum;

    /**
     * The list of keys used to create control programs under the account.<br>
     * Signatures from these keys are required for spending funds held in the account.<br>
     * <strong>Must set with {@link #addRootXpub(String)} or {@link #setRootXpubs(List)} before calling {@link #create(Client)}.</strong>
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
     * @param client client object that makes request to the core
     * @return an account object
     * @throws APIException This exception is raised if the api returns errors while creating the account.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Account create(Client client) throws ChainException {
      return client.singletonBatchRequest(
          "create-account", Arrays.asList(this), Account.class, APIException.class);
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
     * <strong>Must be called before {@link #create(Client)}.</strong>
     * @param quorum proposed quorum
     * @return updated builder object
     */
    public Builder setQuorum(int quorum) {
      this.quorum = quorum;
      return this;
    }

    /**
     * Adds a key to the builder's list.<br>
     * <strong>Either this or {@link #setRootXpubs(List)} must be called before {@link #create(Client)}.</strong>
     * @param xpub key
     * @return updated builder object.
     */
    public Builder addRootXpub(String xpub) {
      this.rootXpubs.add(xpub);
      return this;
    }

    /**
     * Sets the builder's list of keys.<br>
     * <strong>Note:</strong> any existing keys will be replaced.<br>
     * <strong>Either this or {@link #addRootXpub(String)} must be called before {@link #create(Client)}.</strong>
     * @param xpubs list of xpubs
     * @return updated builder object
     */
    public Builder setRootXpubs(List<String> xpubs) {
      this.rootXpubs = new ArrayList<>(xpubs);
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
