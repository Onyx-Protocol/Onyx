package com.chain.api;

import com.chain.http.*;
import com.chain.exception.*;
import com.google.gson.annotations.SerializedName;

import java.util.*;

public class CoreConfig {

  public static class Info {
    public String state;

    @SerializedName("is_configured")
    public boolean isConfigured;

    @SerializedName("configured_at")
    public Date configuredAt;

    @SerializedName("is_signer")
    public boolean isSigner;

    @SerializedName("is_generator")
    public boolean isGenerator;

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

    @SerializedName("crosscore_rpc_version")
    public int crosscoreRpcVersion;

    @SerializedName("core_id")
    public String coreId;

    public String version;

    @SerializedName("build_commit")
    public String buildCommit;

    @SerializedName("build_date")
    public String buildDate;

    @SerializedName("build_config")
    public BuildConfig buildConfig;

    public Health health;

    public Snapshot snapshot;

    public static class BuildConfig {
      @SerializedName("is_loopback_auth")
      public boolean isLoopbackAuth;

      @SerializedName("is_mockhsm")
      public boolean isMockHsm;

      @SerializedName("is_reset")
      public boolean isReset;

      @SerializedName("is_plain_http")
      public boolean isPlainHttp;
    }

    public static class Health {
      public Map<String, String> errors;
    }

    public static class Snapshot {
      public int attempt;

      public long height;

      public long size;

      public long downloaded;

      @SerializedName("in_progress")
      public boolean inProgress;
    }
  }

  public static Info getInfo(Client client) throws ChainException {
    return client.request("info", null, Info.class);
  }

  public static void reset(Client client) throws ChainException {
    client.request("reset", null, SuccessMessage.class);
  }

  public static void resetEverything(Client client) throws ChainException {
    Map<String, Object> params = new HashMap<>();
    params.put("everything", true);
    client.request("reset", params, SuccessMessage.class);
  }

  public static class Builder {
    @SerializedName("is_generator")
    private boolean isGenerator;

    @SerializedName("generator_url")
    private String generatorUrl;

    @SerializedName("generator_access_token")
    private String generatorAccessToken;

    @SerializedName("blockchain_id")
    private String blockchainId;

    public Builder setIsGenerator(boolean isGenerator) {
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
