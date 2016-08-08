package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.chain.signing.KeyHandle;
import com.google.gson.annotations.SerializedName;

import java.util.*;

public class Asset {
    public String id;

    /**
     * The list of public keys attached to the asset
     */
    public List<String> xpubs;

    /**
     * The immutable asset definition
     */
    public Map<String, Object> definition;

    /**
     * User-specified tag structure for the asset
     */
    public Map<String, Object> tags;

    public int quorum;

    public static class Page extends BasePage<Asset> {
        public Page next(Context ctx)
        throws ChainException {
            return ctx.request("list-assets", this.queryPointer, Page.class);
        }
    }

    public static class Query extends BaseQuery<Page> {
        public Page search(Context ctx)
        throws ChainException {
            return ctx.request("list-assets", this.queryPointer, Page.class);
        }
    }

    public static class Builder {
        public Map<String, Object> definition;
        public Map<String, Object> tags;
        public List<String> xpubs;
        public int quorum;
        @SerializedName("client_token")
        private String clientToken;

        public Builder() {
            this.definition = new HashMap<>();
            this.tags = new HashMap<>();
            this.xpubs = new ArrayList<>();
        }

        public Asset create(Context ctx)
        throws ChainException {
            this.clientToken = UUID.randomUUID().toString();
            return ctx.request("create-asset", this, Asset.class);
        }

        public Builder setDefinition(Map<String, Object> definition) {
            this.definition = definition;
            return this;
        }

        public Builder setTags(Map<String, Object> tags) {
            this.tags = tags;
            return this;
        }

        public Builder addTag(String key, Object value) {
            this.tags.put(key, value);
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
    }
}
