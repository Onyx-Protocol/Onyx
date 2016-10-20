package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.http.Client;

import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.exception.JSONException;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Represents the balance of a particular asset (or assets) summed over specified parameters.
 */
public class Balance {
  /**
   * List of parameters on which to sum unspent outputs.
   */
  @SerializedName("sum_by")
  public Map<String, String> sumBy;

  /**
   * Sum of the unspent outputs.
   */
  public long amount;

  /**
   * A paged collection of asset balances returned from a query.
   */
  public static class Items extends PagedItems<Balance> {
    /**
     * Requests a page of asset balances based on an underlying query.
     * @return a collection of balance objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the balances.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public Items getPage() throws ChainException {
      Items items = this.client.request("list-balances", this.next, Items.class);
      items.setClient(this.client);
      return items;
    }
  }

  /**
   * A builder class for generating balance queries.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Executes queries on asset balances.
     * @return a collection of balance objects
     * @throws APIException This exception is raised if the api returns errors while retrieving the balances.
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

    /**
     * Sets the list of unspent output attributes to sum by
     * @param sumBy list of sum by parameters
     * @return updated builder object
     */
    public QueryBuilder setSumBy(List<String> sumBy) {
      this.next.sumBy = new ArrayList<>(sumBy);
      return this;
    }
  }
}
