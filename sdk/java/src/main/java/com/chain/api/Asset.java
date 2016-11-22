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
 * A single asset on a Chain OS blockchain network, capable of being issued and transferred in transactions.
 */
public class Asset {
  /**
   * Globally unique identifier of the asset.<br>
   * Asset version 1 specifies the asset id as the hash of:<br>
   * - the asset version<br>
   * - the asset's issuance program<br>
   * - the core's VM version<br>
   * - the hash of the network's initial block
   */
  public String id;

  /**
   * User specified, unique identifier.
   */
  public String alias;

  /**
   * A program specifying a predicate to be satisfied when issuing the asset.
   */
  @SerializedName("issuance_program")
  public String issuanceProgram;

  /**
   * The list of keys used to create the issuance program for the asset.<br>
   * Signatures from these keys are required for issuing units of the asset.
   */
  public Key[] keys;

  /**
   * The number of keys required to sign an issuance of the asset.
   */
  public int quorum;

  /**
   * User-specified, arbitrary/unstructured data visible across blockchain networks.<br>
   * Version 1 assets specify the definition in their issuance programs, rendering the definition immutable.
   */
  public Map<String, Object> definition;

  /**
   * User-specified, arbitrary/unstructured data local to the asset's originating core.
   */
  public Map<String, Object> tags;

  /**
   * Specifies whether the asset was defined on the local core, or externally.
   */
  @SerializedName("is_local")
  public String isLocal;

  /**
   * Creates a batch of asset objects.<br>
   * <strong>Note:</strong> this method will not throw an exception APIException. Each builder's response object must be checked for error.
   * @param client client object that makes requests to the core
   * @param builders list of asset builders
   * @return a list of asset and/or error objects
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<Asset> createBatch(Client client, List<Builder> builders)
      throws ChainException {
    for (Builder asset : builders) {
      asset.clientToken = UUID.randomUUID().toString();
    }
    return client.batchRequest("create-asset", builders, Asset.class, APIException.class);
  }

  /**
   * A class storing information about the keys associated with the asset.
   */
  public static class Key {
    /**
     * Hex-encoded representation of the root extended public key
     */
    @SerializedName("root_xpub")
    public String rootXpub;

    /**
     * The derived public key, used in the asset's issuance program.
     */
    @SerializedName("asset_pubkey")
    public String assetPubkey;

    /**
     * The derivation path of the derived key.
     */
    @SerializedName("asset_derivation_path")
    public String[] assetDerivationPath;
  }

  /**
   * A paged collection of assets returned from a query.
   */
  public static class Items extends PagedItems<Asset> {
    /**
     * Requests a page of assets based on an underlying query.
     * @return a page of asset objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the assets.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    @Override
    public Items getPage() throws ChainException {
      Items items = this.client.request("list-assets", this.next, Items.class);
      items.setClient(this.client);
      return items;
    }
  }

  /**
   * Asset.QueryBuilder utilizes the builder pattern to create {@link Asset} queries.<br>
   * The possible parameters for each query can be found on the {@link BaseQueryBuilder} class.<br>
   * All parameters are optional, and should be set to filter the results accordingly.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Executes a query on the core's assets.
     * @param client client object that makes requests to the core
     * @return a collection of asset objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the assets.
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
   * Asset.Builder utilizes the builder pattern to create {@link Asset} objects.
   * The following attributes are required to be set: {@link #rootXpubs}, {@link #quorum}.
   */
  public static class Builder {
    /**
     * User specified, unique identifier.
     */
    public String alias;

    /**
     * User-specified, arbitrary/unstructured data visible across blockchain networks.<br>
     * Version 1 assets specify the definition in their issuance programs, rendering the definition immutable.
     */
    public Map<String, Object> definition;

    /**
     * User-specified, arbitrary/unstructured data local to the asset's originating core.
     */
    public Map<String, Object> tags;

    /**
     * The list of keys used to create the issuance program for the asset.<br>
     * Signatures from these keys are required for issuing units of the asset.<br>
     * <strong>Must set with {@link #addRootXpub(String)} or {@link #setRootXpubs(List)} before calling {@link #create(Client)}.</strong>
     */
    @SerializedName("root_xpubs")
    public List<String> rootXpubs;

    /**
     * The number of keys required to sign an issuance of the asset.<br>
     * <strong>Must set with {@link #setQuorum(int)} before calling {@link #create(Client)}.</strong>
     */
    public int quorum;

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
     * Creates an asset object.
     * @param client client object that makes request to the core
     * @return an asset object
     * @throws APIException This exception is raised if the api returns errors while creating the asset.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Asset create(Client client) throws ChainException {
      return client.singletonBatchRequest(
          "create-asset", Arrays.asList(this), Asset.class, APIException.class);
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
     * Adds a field to the existing definition object (initializing the object if it doesn't exist).
     * @param key key of the definition field
     * @param value value of the definition field
     * @return updated builder object
     */
    public Builder addDefinitionField(String key, Object value) {
      if (this.definition == null) {
        this.definition = new HashMap<>();
      }
      this.definition.put(key, value);
      return this;
    }

    /**
     * Sets the asset definition object.<br>
     * <strong>Note:</strong> any existing asset definition fields will be replaced.
     * @param definition asset definition object
     * @return updated builder object
     */
    public Builder setDefinition(Map<String, Object> definition) {
      this.definition = definition;
      return this;
    }

    /**
     * Adds a field to the existing asset tags object (initializing the object if it doesn't exist).
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
     * Sets the asset tags object.<br>
     * <strong>Note:</strong> any existing asset tag fields will be replaced.
     * @param tags asset tags object
     * @return updated builder object
     */
    public Builder setTags(Map<String, Object> tags) {
      this.tags = tags;
      return this;
    }

    /**
     * Sets the quorum of the issuance program.
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
  }
}
