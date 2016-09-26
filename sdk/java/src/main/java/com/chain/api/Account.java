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
   * The number of keys required to signing transactions for the account.
   */
  public int quorum;

  /**
   * The list of keys used to create control programs under the account.<br>
   * Signatures from these keys are required for spending funds held in the account.
   */
  public Key[] keys;

  /**
   * User-specified tag structure for the account
   */
  public Map<String, Object> tags;

  public static class Key {
    @SerializedName("root_xpub")
    public String rootXpub;

    @SerializedName("account_xpub")
    public String accountXpub;

    @SerializedName("account_derivation_path")
    public int[] derivationPath;
  }

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

    @SerializedName("root_xpubs")
    public List<String> rootXpubs;

    public Map<String, Object> tags;

    @SerializedName("client_token")
    private String clientToken;

    public Builder() {
      this.tags = new HashMap<>();
      this.rootXpubs = new ArrayList<>();
    }

    /**
     * Creates an account object.
     *
     * @param ctx context object that makes requests to core
     * @return an account object
     * @throws ChainException
     */
    public Account create(Context ctx) throws ChainException {
      return ctx.singletonBatchRequest("create-account", this, Account.class);
    }

    /**
     * Creates a batch of account objects.
     *
     * @param ctx context object that makes requests to core
     * @param accts list of account builders
     * @return a list of account objects
     * @throws ChainException
     */
    public static List<Account> createBatch(Context ctx, List<Builder> accts) throws ChainException {
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

    public Builder addRootXpub(String xpub) {
      this.rootXpubs.add(xpub);
      return this;
    }

    public Builder setRootXpubs(List<String> xpubs) {
      this.rootXpubs = new ArrayList<>();
      for (String xpub : xpubs) {
        this.rootXpubs.add(xpub);
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
