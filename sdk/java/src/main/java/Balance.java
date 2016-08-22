package com.chain;

import com.google.gson.annotations.SerializedName;

import java.math.BigInteger;
import java.util.List;

public class Balance {
    @SerializedName("group_by")
    public List<String> groupBy;
    public BigInteger amount;

    public static class Items extends PagedItems<Balance> {
        public Items getPage() throws ChainException {
            Items items = this.context.request("list-balances", this.query, Items.class);
            items.setContext(this.context);
            return items;
        }
    }

    public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
        public Items execute(Context ctx) throws ChainException {
            Items items = new Items();
            items.setContext(ctx);
            items.setQuery(this.query);
            return items.getPage();
        }

        public QueryBuilder setTimestamp(long time) {
            this.query.timestamp = time;
            return this;
        }
    }
}
