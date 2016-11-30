package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.http.Client;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;

import com.chain.proto.FilterParam;
import com.chain.proto.ListUnspentOutputsQuery;
import com.chain.proto.ListUnspentOutputsResponse;
import com.google.common.reflect.TypeToken;
import com.google.gson.annotations.SerializedName;

import java.util.List;
import java.util.Map;

public class UnspentOutput {
  /**
   * The type of action being taken on the output.<br>
   * Possible actions are "control_account", "control_program", and "retire".
   */
  public String type;

  /**
   * The purpose of the output.<br>
   * Possible purposes are "receive" and "change". Only populated if the
   * output's control program was generated locally.
   */
  public String purpose;

  /**
   * The ID of the transaction in which the unspent output appears.
   */
  @SerializedName("transaction_id")
  public String transactionId;

  /**
   * The output's position in a transaction's list of outputs.
   */
  public int position;

  /**
   * The id of the asset being controlled.
   */
  @SerializedName("asset_id")
  public String assetId;

  /**
   * The alias of the asset being controlled.
   */
  @SerializedName("asset_alias")
  public String assetAlias;

  /**
   * The definition of the asset being controlled (possibly null).
   */
  @SerializedName("asset_definition")
  public Map<String, Object> assetDefinition;

  /**
   * The tags of the asset being controlled (possibly null).
   */
  @SerializedName("asset_tags")
  public Map<String, Object> assetTags;

  /**
   * A flag indicating whether the asset being controlled is local.
   * Possible values are "yes" or "no".
   */
  @SerializedName("asset_is_local")
  public String assetIsLocal;

  /**
   * The number of units of the asset being controlled.
   */
  public long amount;

  /**
   * The id of the account controlling this output (possibly null if a control program is specified).
   */
  @SerializedName("account_id")
  public String accountId;

  /**
   * The alias of the account controlling this output (possibly null if a control program is specified).
   */
  @SerializedName("account_alias")
  public String accountAlias;

  /**
   * The tags associated with the account controlling this output (possibly null if a control program is specified).
   */
  @SerializedName("account_tags")
  public Map<String, Object> accountTags;

  /**
   * The control program which must be satisfied to transfer this output.
   */
  @SerializedName("control_program")
  public String controlProgram;

  /**
   * User specified, unstructured data embedded within an input (possibly null).
   */
  @SerializedName("reference_data")
  public Map<String, Object> referenceData;

  /**
   * A flag indicating if the output is local.
   * Possible values are "yes" or "no".
   */
  @SerializedName("is_local")
  public String isLocal;

  /**
   * A paged collection of unspent outputs returned from a query.
   */
  public static class Items extends PagedItems<UnspentOutput, ListUnspentOutputsQuery> {
    /**
     * Requests a page of unspent outputs based on an underlying query.
     * @return a collection of unspent output objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the unspent outputs.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     */
    @Override
    public Items getPage() throws ChainException {
      ListUnspentOutputsResponse resp = this.client.app().listUnspentOutputs(this.next);
      if (resp.hasError()) {
        throw new APIException(resp.getError());
      }

      Items items = new Items();
      items.list =
          this.client.deserialize(
              new String(resp.getItems().toByteArray()),
              new TypeToken<List<UnspentOutput>>() {}.getType());
      items.lastPage = resp.getLastPage();
      items.next = resp.getNext();
      items.setClient(this.client);
      return items;
    }

    public void setNext(Query query) {
      ListUnspentOutputsQuery.Builder builder = ListUnspentOutputsQuery.newBuilder();
      if (query.filter != null && !query.filter.isEmpty()) {
        builder.setFilter(query.filter);
      }
      if (query.after != null && !query.after.isEmpty()) {
        builder.setAfter(query.after);
      }
      builder.setTimestamp(query.timestamp);

      if (query.filterParams != null) {
        for (Query.FilterParam param : query.filterParams) {
          builder.addFilterParams(param.toProtobuf());
        }
      }

      this.next = builder.build();
    }
  }

  /**
   * A builder class for generating unspent output queries.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Executes queries on unspent outputs.
     * @return a collection of unspent output objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the unspent outputs.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     */
    public Items execute(Client client) throws ChainException {
      Items items = new Items();
      items.setClient(client);
      items.setNext(this.next);
      return items.getPage();
    }

    /**
     * Sets the latest timestamp for unspent outputs to be included in the results.
     * @param timestampMS timestamp in milliseconds
     * @return updated builder object
     */
    public QueryBuilder setTimestamp(long timestampMS) {
      this.next.timestamp = timestampMS;
      return this;
    }
  }
}
