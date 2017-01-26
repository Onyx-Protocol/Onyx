package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.http.Client;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;

import com.google.gson.annotations.SerializedName;

import java.util.Map;

public class UnspentOutput {
  /**
   * The ID of the output.
   */
  @SerializedName("id")
  public String id;

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
  public static class Items extends PagedItems<UnspentOutput> {
    /**
     * Requests a page of unspent outputs based on an underlying query.
     * @return a collection of unspent output objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the unspent outputs.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items getPage() throws ChainException {
      Items items = this.client.request("list-unspent-outputs", this.next, Items.class);
      items.setClient(this.client);
      return items;
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
     * @throws JSONException This exception is raised due to malformed json requests or responses.
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
