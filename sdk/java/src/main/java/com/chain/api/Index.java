package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;

import java.util.HashMap;
import java.util.Map;

public class Index {
    public String id;
    public String type;
    public String query;
    public Boolean unspents;

    public static class Page extends BasePage<Index> {
        public Page next(Context ctx)
        throws ChainException {
            return ctx.request("list-indexes", this.queryPointer, Page.class);
        }
    }

    public static class Query extends BaseQuery<Page> {
        public Page search(Context ctx)
        throws ChainException {
            return ctx.request("list-indexes", this.queryPointer, Page.class);
        }

        public Index find(Context ctx, String id)
        throws ChainException {
            Map<String, Object> req = new HashMap<>();
            req.put("id", id);
            return ctx.request("get-index", req, Index.class);
        }
    }

    public static class Builder {
        public String id;
        public String type;
        public String query;
        public boolean unspents;

        public Index create(Context ctx)
        throws ChainException {
            return ctx.request("create-index", this, Index.class);
        }

        public Builder setId(String id) {
            this.id = id;
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
