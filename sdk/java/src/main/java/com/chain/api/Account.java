package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;
import com.chain.http.*;
import com.chain.proto.*;
import com.google.gson.annotations.SerializedName;
import com.google.protobuf.ByteString;

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
    public byte[] rootXpub;

    /**
     * The extended public key used to create control programs for the account.
     */
    @SerializedName("account_xpub")
    public byte[] accountXpub;

    /**
     * The derivation path of the extended key.
     */
    @SerializedName("account_derivation_path")
    public byte[][] accountDerivationPath;

    private Key(com.chain.proto.Account.Key proto) {
      this.rootXpub = proto.getRootXpub().toByteArray();
      this.accountXpub = proto.getAccountXpub().toByteArray();
      this.accountDerivationPath = new byte[proto.getAccountDerivationPathCount()][];
      for (int i = 0; i < proto.getAccountDerivationPathCount(); i++) {
        this.accountDerivationPath[i] = proto.getAccountDerivationPath(i).toByteArray();
      }
    }

    private static Key[] fromProtobuf(List<com.chain.proto.Account.Key> protos) {
      Key[] resp = new Key[protos.size()];
      for (int i = 0; i < protos.size(); i++) {
        resp[i] = new Key(protos.get(i));
      }
      return resp;
    }
  }

  private Account(com.chain.proto.Account proto, Client client) {
    this.id = proto.getId();
    this.alias = proto.getAlias();
    this.quorum = proto.getQuorum();
    this.keys = Key.fromProtobuf(proto.getKeysList());
    if (proto.getTags() != null && !proto.getTags().isEmpty()) {
      String tags = new String(proto.getTags().toByteArray());
      this.tags = client.deserialize(tags);
    }
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
    ArrayList<CreateAccountsRequest.Request> reqs = new ArrayList();
    for (Builder builder : builders) {
      CreateAccountsRequest.Request.Builder req = CreateAccountsRequest.Request.newBuilder();
      req.setQuorum(builder.quorum);
      req.setClientToken(UUID.randomUUID().toString());
      if (builder.alias != null && !builder.alias.isEmpty()) {
        req.setAlias(builder.alias);
      }
      if (builder.rootXpubs != null && !builder.rootXpubs.isEmpty()) {
        req.addAllRootXpubs(builder.rootXpubs);
      }
      if (builder.tags != null && !builder.tags.isEmpty()) {
        req.setTags(ByteString.copyFrom(client.serialize(builder.tags)));
      }
      reqs.add(req.build());
    }
    CreateAccountsRequest req = CreateAccountsRequest.newBuilder().addAllRequests(reqs).build();
    CreateAccountsResponse resp = client.app().createAccounts(req);

    if (resp.hasError()) {
      throw new APIException(resp.getError());
    }

    Map<Integer, Account> successes = new LinkedHashMap();
    Map<Integer, APIException> errors = new LinkedHashMap();

    for (int i = 0; i < resp.getResponsesCount(); i++) {
      CreateAccountsResponse.Response r = resp.getResponses(i);
      if (r.hasError()) {
        errors.put(i, new APIException(r.getError()));
      } else {
        successes.put(i, new Account(r.getAccount(), client));
      }
    }

    return new BatchResponse<Account>(successes, errors);
  }

  /**
   * A paged collection of accounts returned from a query.
   */
  public static class Items extends PagedItems<Account, ListAccountsQuery> {
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
      ListAccountsResponse resp = this.client.app().listAccounts(this.next);
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      Items items = new Items();
      for (com.chain.proto.Account account : resp.getItemsList()) {
        items.list.add(new Account(account, client));
      }
      items.lastPage = resp.getLastPage();
      items.next = resp.getNext();
      items.setClient(this.client);
      return items;
    }

    public void setNext(Query query) {
      ListAccountsQuery.Builder builder = ListAccountsQuery.newBuilder();
      if (query.filter != null && !query.filter.isEmpty()) {
        builder.setFilter(query.filter);
      }
      if (query.after != null && !query.filter.isEmpty()) {
        builder.setAfter(query.after);
      }

      if (query.filterParams != null) {
        for (Query.FilterParam param : query.filterParams) {
          builder.addFilterParams(param.toProtobuf());
        }
      }

      this.next = builder.build();
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
    public List<ByteString> rootXpubs;

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
     */
    public Account create(Client client) throws ChainException {
      BatchResponse<Account> resp = Account.createBatch(client, Arrays.asList(this));
      if (resp.isError(0)) {
        throw resp.errorsByIndex().get(0);
      }
      return resp.successesByIndex().get(0);
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
      return addRootXpub(Util.hexStringToByteArray(xpub));
    }

    public Builder addRootXpub(byte[] xpub) {
      this.rootXpubs.add(ByteString.copyFrom(xpub));
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
      this.rootXpubs = new ArrayList();
      for (String xpub : xpubs) {
        this.rootXpubs.add(ByteString.copyFrom(Util.hexStringToByteArray(xpub)));
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
