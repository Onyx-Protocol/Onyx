package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.Map;

public class Balance {
    public static class Page extends BasePage<Map<String, Object>> {
        public Page next(Context ctx)
        throws ChainException {
            return ctx.request("list-balances", this.queryPointer, Page.class);
        }
    }

    public static class Query {
        @SerializedName("query")
        protected QueryPointer queryPointer;

        public Query() {
            this.queryPointer = new QueryPointer();
        }

        public Query useIndex(String index) {
            this.queryPointer.index = index;
            return this;
        }

        public Query withChQL(String chql) {
            this.queryPointer.chql = chql;
            return this;
        }

        public Query addParameter(String param) {
            this.queryPointer.params.add(param);
            return this;
        }

        public Query setParameters(ArrayList<String> params) {
            this.queryPointer.params = new ArrayList<>();
            for (String param : params) {
                this.queryPointer.params.add(param);
            }
            return this;
        }

        public Page listBalances(Context ctx)
        throws ChainException {
            return ctx.request("list-balances", this.queryPointer, Page.class);
        }

        public Query setTimestamp(long time) {
            this.queryPointer.timestamp = time;
            return this;
        }
    }
}