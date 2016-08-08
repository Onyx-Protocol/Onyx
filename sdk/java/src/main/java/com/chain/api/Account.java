package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.chain.signing.KeyHandle;
import com.google.gson.annotations.SerializedName;

import java.util.*;

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
    public List<String> xpubs;

    /**
     * User-specified tag structure for the account
     */
    public Map<String, Object> tags;

    /**
     *
     */

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
        private Map<String, Object> tags;
        @SerializedName("client_token")
        private String clientToken;

        public Builder() {
            this.xpubs = new ArrayList<>();
        }

        public Account create(Context ctx)
        throws ChainException {
            this.clientToken = UUID.randomUUID().toString();
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

        public Builder addXpub(KeyHandle key) {
            this.xpubs.add(key.getXPub());
            return this;
        }

        public Builder setXpubs(List<KeyHandle> keys) {
            this.xpubs = new ArrayList<>();
            for (KeyHandle key : keys) {
                this.xpubs.add(key.getXPub());
            }
            return this;
        }

        public Builder setTags(Map<String, Object> tags) {
            this.tags = tags;
            return this;
        }
    }
}
