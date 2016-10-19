package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Client;

import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class Balance {
  @SerializedName("sum_by")
  public Map<String, String> sumBy;

  public long amount;

  public static class Items extends PagedItems<Balance> {
    public Items getPage() throws ChainException {
      Items items = this.client.request("list-balances", this.next, Items.class);
      items.setClient(this.client);
      return items;
    }
  }

  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    public Items execute(Client client) throws ChainException {
      Items items = new Items();
      items.setClient(client);
      items.setNext(this.next);
      return items.getPage();
    }

    public QueryBuilder setTimestamp(long time) {
      this.next.timestamp = time;
      return this;
    }

    public QueryBuilder setSumBy(List<String> sumBy) {
      this.next.sumBy = new ArrayList<>(sumBy);
      return this;
    }
  }
}
