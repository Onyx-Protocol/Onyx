package com.chain.api;

import com.chain.exception.APIException;
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
     * Unique account identifier
     */
    public String id;

    /**
     * User specified, unique identifier
     */
    public String alias;

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

    // Error data
    public String code;
    public String message;
    public String detail;

    public static class Items extends PagedItems<Account> {
        public Items getPage() throws ChainException {
            Items items = this.context.request("list-accounts", this.query, Items.class);
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
    }

    public static class Builder {
        public String alias;
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
            Account result = accts.get(0);
            if (result.id == null) {
                throw new APIException(
                    result.code,
                    result.message,
                    result.detail,
                    null
                );
            }
            return result;
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

        public Builder setAlias(String alias) {
            this.alias = alias;
            return this;
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
