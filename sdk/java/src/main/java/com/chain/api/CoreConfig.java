package com.chain.api;

import com.chain.http.*;
import com.chain.exception.*;
import com.google.gson.annotations.SerializedName;

import java.util.*;

public class CoreConfig {

  public static class Info {
    public String state;

    /**
     * Whether the core has been configured.
     */
    @SerializedName("is_configured")
    public boolean isConfigured;

    /**
     * Date reflecting when the core was configured.
     */
    @SerializedName("configured_at")
    public Date configuredAt;

    /**
     * Whether the core is configured as a block signer.
     */
    @SerializedName("is_signer")
    public boolean isSigner;

    /**
     * Whether the core is configured as the blockchain generator.
     */
    @SerializedName("is_generator")
    public boolean isGenerator;

    /**
     * URL of the generator.
     */
    @SerializedName("generator_url")
    public String generatorUrl;

    /**
     * The access token used to connect to the generator.
     */
    @SerializedName("generator_access_token")
    public String generatorAccessToken;

    /**
     * Hash of the initial block.
     */
    @SerializedName("blockchain_id")
    public String blockchainId;

    /**
     * Height of the blockchain in the local core.
     */
    @SerializedName("block_height")
    public long blockHeight;

    /**
     * Height of the blockchain in the generator.
     */
    @SerializedName("generator_block_height")
    public long generatorBlockHeight;

    /**
     * Date reflecting the last time generator_block_height was updated.
     */
    @SerializedName("generator_block_height_fetched_at")
    public Date generatorBlockHeightFetchedAt;

    /**
     * The cross-core API version supported by this core.
     */
    @SerializedName("crosscore_rpc_version")
    public int crosscoreRpcVersion;

    /**
     * A random identifier for the core, generated during configuration.
     */
    @SerializedName("core_id")
    public String coreId;

    /**
     * The release version of the cored binary.
     */
    public String version;

    /**
     * Git SHA of build source.
     */
    @SerializedName("build_commit")
    public String buildCommit;

    /**
     * Unixtime (as string) of binary build.
     */
    @SerializedName("build_date")
    public String buildDate;

    /**
     * Features enabled or disabled in this build of Chain Core.
     */
    @SerializedName("build_config")
    public BuildConfig buildConfig;

    /**
     * Blockchain error information.
     */
    public Health health;

    public Snapshot snapshot;

    public static class BuildConfig {
      /**
       * Whether any request from the loopback device (localhost) should be
       * automatically authenticated and authorized, without additional
       * credentials.
       */
      @SerializedName("is_localhost_auth")
      public boolean isLocalhostAuth;

      /**
       * Whether the MockHSM API is enabled.
       */
      @SerializedName("is_mockhsm")
      public boolean isMockHsm;

      /**
       * Whether the core reset API call is enabled.
       */
      @SerializedName("is_reset")
      public boolean isReset;

      /**
       * Whether non-TLS HTTP requests (http://...) are allowed.
       */
      @SerializedName("is_http_ok")
      public boolean isHttpOk;
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

  /**
   * Gets info on specified Chain Core.
   * @param client client object that makes requests to the core
   * @return Info
   */
  public static Info getInfo(Client client) throws ChainException {
    return client.request("info", null, Info.class);
  }

  /**
   * Resets specified Chain Core, preserving access tokens and MockHSM keys.
   * @param client client object that makes requests to the core
   */
  public static void reset(Client client) throws ChainException {
    client.request("reset", null, SuccessMessage.class);
  }

  /**
   * Resets all data on the specified Chain Core, including access tokens
   * and MockHSM keys.
   * @param client client object that makes requests to the core
   */
  public static void resetEverything(Client client) throws ChainException {
    Map<String, Object> params = new HashMap<>();
    params.put("everything", true);
    client.request("reset", params, SuccessMessage.class);
  }

  public static class Builder {
    /**
     * Whether the core is configured as the blockchain generator.
     */
    @SerializedName("is_generator")
    private boolean isGenerator;

    /**
     * URL of the generator.
     */
    @SerializedName("generator_url")
    private String generatorUrl;

    /**
     * The access token used to connect to the generator.
     */
    @SerializedName("generator_access_token")
    private String generatorAccessToken;

    /**
     * Hash of the initial block.
     */
    @SerializedName("blockchain_id")
    private String blockchainId;

    /**
     * Sets whether the core is a block generator.
     * @param isGenerator boolean indicating generator status
     * @return updated builder object
     */
    public Builder setIsGenerator(boolean isGenerator) {
      this.isGenerator = isGenerator;
      return this;
    }

    /**
     * Sets the URL of the remote block generator.
     * @param url the URL of the remote generator
     * @return updated builder object
     */
    public Builder setGeneratorUrl(String url) {
      generatorUrl = url;
      return this;
    }

    /**
     * Sets the access token for the remote block generator.
     * @param token an acces stoken
     * @return updated builder object
     */
    public Builder setGeneratorAccessToken(String token) {
      generatorAccessToken = token;
      return this;
    }

    /**
     * Sets the remote blockchain id.
     * @param blockchainId an initial block hash
     * @return updated builder object
     */
    public Builder setBlockchainId(String blockchainId) {
      this.blockchainId = blockchainId;
      return this;
    }

    /**
     * Configures specified Chain Core.
     * @param client client object that makes requests to the core
     */
    public void configure(Client client) throws ChainException {
      client.request("configure", this);
    }
  }
}
