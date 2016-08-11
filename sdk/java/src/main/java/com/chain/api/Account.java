package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
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

    public Account setTags(Map<String, Object> tags) {
        this.tags = tags;
        return this;
    }

    public Account addTag(String key, Object value) {
        this.tags.put(key, value);
        return this;
    }

    public Account removeTag(String key) {
        this.tags.remove(key);
        return this;
    }

    public Account updateTags(Context ctx) throws ChainException {
        HashMap<String, Object> requestBody = new HashMap<>();
        requestBody.put("account_id", this.id);
        requestBody.put("tags", this.tags);
        return ctx.request("set-account-tags", requestBody, Account.class);
    }

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

    public static class Query extends BaseQuery<Query> {
        public Page search(Context ctx)
        throws ChainException {
            return ctx.request("list-accounts", this.queryPointer, Page.class);
        }
    }

    public static class Builder {
        public int quorum;
        public List<String> xpubs;
        public Map<String, Object> tags;
        @SerializedName("client_token")
        private String clientToken;

        public Builder() {
            this.tags = new HashMap<>();
            this.xpubs = new ArrayList<>();
        }

        /**
         * Creates an account object.
         *
         * @param ctx context object that makes requests to core
         * @return an account object
         * @throws ChainException
         */
        public Account create(Context ctx)
        throws ChainException {
            List<Account> accts = Account.Builder.create(ctx, Arrays.asList(this));
            return accts.get(0);
        }

        /**
         * Creates a batch of account objects.
         *
         * @param ctx context object that makes requests to core
         * @return an account object
         * @throws ChainException
         */
        public static List<Account> create(Context ctx, List<Builder> accts)
        throws ChainException {
            for (Builder acct : accts) {
                acct.clientToken = UUID.randomUUID().toString();
            }
            Type type = new TypeToken<List<Account>>() {}.getType();
            return ctx.request("create-account", accts, type);
        }

        public Builder setQuorum(int quorum) {
            this.quorum = quorum;
            return this;
        }

        public Builder addXpub(String xpub) {
            this.xpubs.add(xpub);
            return this;
        }

        public Builder setXpubs(List<String> xpubs) {
            this.xpubs = new ArrayList<>();
            for (String xpub : xpubs) {
                this.xpubs.add(xpub);
            }
            return this;
        }

        public Builder addTag(String key, Object value) {
            this.tags.put(key, value);
            return this;
        }

        public Builder setTags(Map<String, Object> tags) {
            this.tags = tags;
            return this;
        }
    }
}
