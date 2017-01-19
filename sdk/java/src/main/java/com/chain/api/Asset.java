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
  public byte[] id;

  /**
   * User specified, unique identifier.
   */
  public String alias;

  /**
   * A program specifying a predicate to be satisfied when issuing the asset.
   */
  @SerializedName("issuance_program")
  public byte[] issuanceProgram;

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
  public boolean isLocal;

  private Asset(com.chain.proto.Asset proto, Client client) {
    this.id = proto.getId().toByteArray();
    this.alias = proto.getAlias();
    this.issuanceProgram = proto.getIssuanceProgram().toByteArray();
    this.keys = Key.fromProtobuf(proto.getKeysList());
    this.quorum = proto.getQuorum();
    if (proto.getDefinition() != null && !proto.getDefinition().isEmpty()) {
      String definition = new String(proto.getDefinition().toByteArray());
      this.definition = client.deserialize(definition);
    }
    if (proto.getTags() != null && !proto.getTags().isEmpty()) {
      String tags = new String(proto.getTags().toByteArray());
      this.tags = client.deserialize(tags);
    }
    this.isLocal = proto.getIsLocal();
  }

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
    ArrayList<CreateAssetsRequest.Request> reqs = new ArrayList();
    for (Builder builder : builders) {
      CreateAssetsRequest.Request.Builder req = CreateAssetsRequest.Request.newBuilder();
      req.setQuorum(builder.quorum);
      req.setClientToken(UUID.randomUUID().toString());
      if (builder.alias != null && !builder.alias.isEmpty()) {
        req.setAlias(builder.alias);
      }
      if (builder.rootXpubs != null && !builder.rootXpubs.isEmpty()) {
        req.addAllRootXpubs(builder.rootXpubs);
      }

      if (builder.definition != null && !builder.definition.isEmpty()) {
        req.setDefinition(ByteString.copyFrom(client.serialize(builder.definition)));
      }

      if (builder.tags != null && !builder.tags.isEmpty()) {
        req.setTags(ByteString.copyFrom(client.serialize(builder.tags)));
      }

      reqs.add(req.build());
    }
    CreateAssetsRequest req = CreateAssetsRequest.newBuilder().addAllRequests(reqs).build();
    CreateAssetsResponse resp = client.app().createAssets(req);

    if (resp.hasError()) {
      throw new APIException(resp.getError());
    }

    Map<Integer, Asset> successes = new LinkedHashMap();
    Map<Integer, APIException> errors = new LinkedHashMap();

    for (int i = 0; i < resp.getResponsesCount(); i++) {
      CreateAssetsResponse.Response r = resp.getResponses(i);
      if (r.hasError()) {
        errors.put(i, new APIException(r.getError()));
      } else {
        successes.put(i, new Asset(r.getAsset(), client));
      }
    }

    return new BatchResponse<Asset>(successes, errors);
  }

  /**
   * A class storing information about the keys associated with the asset.
   */
  public static class Key {
    /**
     * Hex-encoded representation of the root extended public key
     */
    @SerializedName("root_xpub")
    public byte[] rootXpub;

    /**
     * The derived public key, used in the asset's issuance program.
     */
    @SerializedName("asset_pubkey")
    public byte[] assetPubkey;

    /**
     * The derivation path of the derived key.
     */
    @SerializedName("asset_derivation_path")
    public byte[][] assetDerivationPath;

    private Key(com.chain.proto.Asset.Key proto) {
      this.rootXpub = proto.getRootXpub().toByteArray();
      this.assetPubkey = proto.getAssetPubkey().toByteArray();
      this.assetDerivationPath = new byte[proto.getAssetDerivationPathCount()][];
      for (int i = 0; i < proto.getAssetDerivationPathCount(); i++) {
        this.assetDerivationPath[i] = proto.getAssetDerivationPath(i).toByteArray();
      }
    }

    private static Key[] fromProtobuf(List<com.chain.proto.Asset.Key> protos) {
      Key[] resp = new Key[protos.size()];
      for (int i = 0; i < protos.size(); i++) {
        resp[i] = new Key(protos.get(i));
      }
      return resp;
    }
  }

  /**
   * A paged collection of assets returned from a query.
   */
  public static class Items extends PagedItems<Asset, ListAssetsQuery> {
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
      ListAssetsResponse resp = this.client.app().listAssets(this.next);
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      Items items = new Items();
      for (com.chain.proto.Asset asset : resp.getItemsList()) {
        items.list.add(new Asset(asset, client));
      }
      items.lastPage = resp.getLastPage();
      items.next = resp.getNext();
      items.setClient(this.client);
      return items;
    }

    public void setNext(Query query) {
      ListAssetsQuery.Builder builder = ListAssetsQuery.newBuilder();

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
    public List<ByteString> rootXpubs;

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
      BatchResponse<Asset> resp = Asset.createBatch(client, Arrays.asList(this));
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
  }
}
