package com.chain.api;

import com.chain.exception.*;
import com.chain.http.*;
import com.google.gson.annotations.SerializedName;

import java.util.*;

/**
 * Access tokens are used to authenticate requests made to an instance of
 * Chain Core.
 * <p>
 * After creating an access token, you should use an {@link AuthorizationGrant}
 * to assign access policies to the token.
 */
public class AccessToken {
  /**
   * Unique, user-supplied identifier for the access token.
   */
  public String id;

  /**
   * The full value of the access token. Use this when configuring SDK clients
   * for access to Chain Core.
   */
  public String token;

  /**
   * The time at which the access token was created.
   */
  @SerializedName("created_at")
  public Date createdAt;

  /**
   * An interface for iterating the results of an access token query.
   */
  public static class Items extends PagedItems<AccessToken> {
    /**
     * Requests a page of access tokens based on an underlying query.
     * @return a page of access tokens
     * @throws ChainException
     */
    public Items getPage() throws ChainException {
      Items items = this.client.request("list-access-tokens", this.next, Items.class);
      items.setClient(this.client);
      return items;
    }
  }

  /**
   * Starts a new query for access tokens. Use this class to retrieve a list of
   * access tokens recognized by the Chain Core instance.
   */
  public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
    /**
     * Retrieves query results from Chain Core.
     * @param client the client object providing connectivity to the Chain Core instance
     * @return a collection of access tokens
     * @throws ChainException
     */
    public Items execute(Client client) throws ChainException {
      Items items = new Items();
      items.setClient(client);
      items.setNext(this.next);
      return items.getPage();
    }
  }

  /**
   * Sets up an API call for creating access tokens. The {@link #id} property is required.
   */
  public static class Builder {
    /**
     * User specified, unique identifier.
     */
    private String id;

    /**
     * Unique identifier used for request idempotence.
     */
    @SerializedName("client_token")
    private String clientToken;

    /**
     * Creates a new access token.
     * @param client the client object providing connectivity to the Chain Core instance
     * @return an access token object
     * @throws ChainException
     */
    public AccessToken create(Client client) throws ChainException {
      clientToken = UUID.randomUUID().toString();
      return client.request("create-access-token", this, AccessToken.class);
    }

    /**
     * Sets the unique identifier for the new access token.
     * @param id an access token identifier
     * @return updated builder object
     */
    public Builder setId(String id) {
      this.id = id;
      return this;
    }
  }
}
