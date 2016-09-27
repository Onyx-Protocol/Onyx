package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
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
      Items items = this.context.request("list-balances", this.next, Items.class);
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

    public QueryBuilder setSumBy(List<String> sumBy) {
      this.next.sumBy = new ArrayList<>(sumBy);
      return this;
    }
  }
}
