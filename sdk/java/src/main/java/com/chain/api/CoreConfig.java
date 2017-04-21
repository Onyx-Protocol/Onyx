package com.chain.api;

import com.chain.http.*;
import com.chain.exception.*;
import com.google.gson.annotations.SerializedName;

import java.util.*;

public class CoreConfig {

    public static class Info {
        public String state;

        @SerializedName("is_configured")
        public Boolean isConfigured;

        @SerializedName("configured_at")
        public Date configuredAt;

        @SerializedName("is_signer")
        public Boolean isSigner;

        @SerializedName("is_generator")
        public Boolean isGenerator;

        @SerializedName("generator_url")
        public String generatorUrl;

        @SerializedName("generator_access_token")
        public String generatorAccessToken;

        @SerializedName("blockchain_id")
        public String blockchainId;

        @SerializedName("block_height")
        public long blockHeight;

        @SerializedName("generator_block_height")
        public long generatorBlockHeight;

        @SerializedName("generator_block_height_fetched_at")
        public Date generatorBlockHeightFetchedAt;

        @SerializedName("network_rpc_version")
        public int networkRpcVersion;

        @SerializedName("core_id")
        public String coreId;

        public String version;

        @SerializedName("build_commit")
        public String buildCommit;

        @SerializedName("build_date")
        public String buildDate;

        @SerializedName("build_config")
        public BuildConfig buildConfig;

        public Map<String, Object> health;

        public Snapshot snapshot;

        public static class BuildConfig {
            @SerializedName("is_loopback_auth")
            public Boolean isLoopbackAuth;

            @SerializedName("is_mockhsm")
            public Boolean isMockHsm;

            @SerializedName("is_reset")
            public Boolean isReset;
        }

        public static class Snapshot {
            public int attempt;

            public long height;

            public long size;

            public long downloaded;

            @SerializedName("in_progress")
            public Boolean inProgress;
        }
    }

    public static Info getInfo(Client client) throws ChainException {
        return client.request("info", null, Info.class);
    }

    public static void reset(Client client) throws ChainException {
        client.request("reset", null, SuccessMessage.class);
    }

    public static void resetEverything(Client client) throws ChainException {
        Map<String, Boolean> params = new HashMap<>();
        params.put("everything", true);
        client.request("reset", params, SuccessMessage.class);
    }

    public static class Builder {
        @SerializedName("is_generator")
        private Boolean isGenerator;

        @SerializedName("generator_url")
        private String generatorUrl;

        @SerializedName("generator_access_token")
        private String generatorAccessToken;

        @SerializedName("blockchain_id")
        private String blockchainId;

        public Builder setIsGenerator(Boolean isGenerator) {
            this.isGenerator = isGenerator;
            return this;
        }

        public Builder setGeneratorUrl(String url) {
            generatorUrl = url;
            return this;
        }

        public Builder setGeneratorAccessToken(String token) {
            generatorAccessToken = token;
            return this;
        }

        public Builder setBlockchainId(String blockchainId) {
            this.blockchainId = blockchainId;
            return this;
        }

        public void configure(Client client) throws ChainException {
            client.request("configure", this);
        }
    }
}
