package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.util.*;

public class Asset {
    public String id;

    @SerializedName("issuance_program")
    public byte[] issuanceProgram;

    /**
     * The immutable asset definition
     */
    public Map<String, Object> definition;

    /**
     * User-specified tag structure for the asset
     */
    public Map<String, Object> tags;

    public Asset setTags(Map<String, Object> tags) {
        this.tags = tags;
        return this;
    }

    public Asset addTag(String key, Object value) {
        this.tags.put(key, value);
        return this;
    }

    public Asset removeTag(String key) {
        this.tags.remove(key);
        return this;
    }

    public Asset updateTags(Context ctx) throws ChainException {
        HashMap<String, Object> requestBody = new HashMap<>();
        requestBody.put("asset_id", this.id);
        requestBody.put("tags", this.tags);
        return ctx.request("set-asset-tags", requestBody, Asset.class);
    }

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
            List<Asset> assets = Asset.Builder.create(ctx, Arrays.asList(this));
            return assets.get(0);
        }

        public static List<Asset> create(Context ctx, List<Builder> assets)
        throws ChainException {
            for (Builder asset : assets) {
                asset.clientToken = UUID.randomUUID().toString();
            }
            Type type = new TypeToken<List<Asset>>() {}.getType();
            return ctx.request("create-asset", assets, type);
        }

        public Builder setDefinition(Map<String, Object> definition) {
            this.definition = definition;
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
    }
}
