package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class Balance {
    @SerializedName("group_by")
    public List<String> groupBy;
    public Integer amount;

    public static class Page extends BasePage<Balance> {
        public Page next(Context ctx)
        throws ChainException {
            return ctx.request("list-balances", this.query, Page.class);
        }
    }

    public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
        public Page execute(Context ctx)
        throws ChainException {
            return ctx.request("list-balances", this.query, Page.class);
        }

        public QueryBuilder setTimestamp(long time) {
            this.query.timestamp = time;
            return this;
        }
    }
}