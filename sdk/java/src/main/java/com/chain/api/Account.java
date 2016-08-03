package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.chain.signing.KeyHandle;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * A single Account on the Chain Core, capable of spending or receiving assets in a transaction
 */
public class Account {
    /**
     * Unique account identifier, optionally user defined
     */
    public String id;

    /**
     * The number of public keys required to signing transactions for the account
     */
    public int quorum;

    /**
     * The list of public keys attached to the account
     */
    @SerializedName("xpubs")
    public List<String> xpubs;

    /**
     * List of user-specified tags on the object
     */
    public List<String> tags;

    /**
     * A single page of Account objects returned from a search query, with a pointer to the next page of results
     * if applicable.
     */
    public static class Page extends BasePage<Account> {
        /**
         *
         *
         * @param ctx
         * @return The next Account.Page of results for the originating query
         * @throws ChainException
         */
        public Page next(Context ctx)
        throws ChainException {
            return ctx.request("list-accounts", this.queryPointer, Page.class);
        }
    }

    public static class Query extends BaseQuery<Page> {
        public Page search(Context ctx)
        throws ChainException {
            return ctx.request("list-accounts", this.queryPointer, Page.class);
        }

        public Account find(Context ctx, String accountId)
        throws ChainException {
            Map<String, Object> req = new HashMap<>();
            req.put("id", accountId);
            return ctx.request("get-account", req, Account.class);
        }
    }

    public static class Builder {
        private String id;
        private int quorum;
        private List<String> xpubs;
        private List<String> tags;

        public Builder() {
            this.xpubs = new ArrayList<>();
            this.tags = new ArrayList<>();
        }

        public Account create(Context ctx)
        throws ChainException {
            return ctx.request("create-account", this, Account.class);
        }

        public Builder setId(String id) {
            this.id = id;
            return this;
        }

        public Builder setQuorum(int quorum) {
            this.quorum = quorum;
            return this;
        }

        public Builder addKey(KeyHandle key) {
            this.xpubs.add(key.getXPub());
            return this;
        }

        public Builder setKey(List<KeyHandle> keys) {
            this.xpubs = new ArrayList<>();
            for (KeyHandle key : keys) {
                this.xpubs.add(key.getXPub());
            }
            return this;
        }

        public Builder addTag(String tag) {
            this.tags.add(tag);
            return this;
        }

        public Builder setTags(List<String> tags) {
            this.tags = tags;
            return this;
        }
    }
}
