package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

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

  public static class Items extends PagedItems<UnspentOutput> {
    public Items getPage() throws ChainException {
      Items items = this.context.request("list-unspent-outputs", this.next, Items.class);
      items.setContext(this.context);
      return items;
    }
  }

  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    public Items execute(Context ctx) throws ChainException {
      Items items = new Items();
      items.setContext(ctx);
      items.setNext(this.next);
      return items.getPage();
    }

    public QueryBuilder setTimestamp(long time) {
      this.next.timestamp = time;
      return this;
    }
  }
}
