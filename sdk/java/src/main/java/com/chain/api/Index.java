package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;

import java.util.HashMap;
import java.util.Map;

public class Index {
    public String id;
    public String alias;
    public String type;
    public String query;
    public Boolean unspents;

    public static class Items extends PagedItems<Index> {
        public Items getPage() throws ChainException {
            Items items = this.context.request("list-indexes", this.query, Items.class);
            items.setContext(this.context);
            return items;
        }
    }

    public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
        public Items execute(Context ctx) throws ChainException {
            Items items = new Items();
            items.setContext(ctx);
            return items.getPage();
        }

        public Index find(Context ctx, String id) throws ChainException {
            Map<String, Object> req = new HashMap<>();
            req.put("id", id);
            return ctx.request("get-index", req, Index.class);
        }
    }

    public static class Builder {
        public String alias;
        public String type;
        public String query;
        public boolean unspents;

        public Index create(Context ctx) throws ChainException {
            return ctx.request("create-index", this, Index.class);
        }

        public Builder setAlias(String alias) {
            this.alias = alias;
            return this;
        }

        public Builder setType(String type) {
            this.type = type;
            return this;
        }

        public Builder setQuery(String query) {
            this.query = query;
            return this;
        }

        public Builder setUnspents(Boolean unspents) {
            this.unspents = unspents;
            return this;
        }
    }
}
