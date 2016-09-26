package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.ChainException;
import com.chain.http.Context;

import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * When used in conjunction with /list-transactions, Cursors can be used to
 * receive notifications about transactions.
 */
public class Cursor {
  /**
   * Cursor ID, automatically generated when a cursor is created.
   */
  public String id;

  /**
   * An optional, user-supplied alias that can be used to uniquely identify
   * this cursor.
   */
  public String alias;

  /**
   * The query filter used in /list-transactions.
   */
  public String filter;

  /**
   * The direction ("asc" or "desc") that this cursor moves through the
   * transaction list. Only "asc" (oldest transactions first) is supported
   * currently.
   */
  public String order;

  /**
   * Indicates the last transaction consumed by this cursor.
   */
  public String after;

  /**
   * Creates a cursor.
   *
   * @param ctx context object that makes requests to core
   * @param alias an alias which uniquely identifies this cursor
   * @param filter a query filter which identifies which transactions this cursor consumes
   * @return a cursor object
   * @throws ChainException
   */
  public static Cursor create(Context ctx, String alias, String filter) throws ChainException {
    Map<String, Object> req = new HashMap<>();
    req.put("alias", alias);
    req.put("filter", filter);
    req.put("client_token", UUID.randomUUID().toString());
    return ctx.request("create-cursor", req, Cursor.class);
  }

  /**
   * Retrieves a cursor by ID.
   *
   * @param ctx context object that makes requests to core
   * @param id the cursor id
   * @return a cursor object
   * @throws ChainException
   */
  public static Cursor getByID(Context ctx, String id) throws ChainException {
    Map<String, Object> req = new HashMap<>();
    req.put("id", id);
    return ctx.request("get-cursor", req, Cursor.class);
  }

  /**
   * Retrieves a cursor by alias.
   *
   * @param ctx context object that makes requests to core
   * @param alias the cursor alias
   * @return a cursor object
   * @throws ChainException
   */
  public static Cursor getByAlias(Context ctx, String alias) throws ChainException {
    Map<String, Object> req = new HashMap<>();
    req.put("alias", alias);
    return ctx.request("get-cursor", req, Cursor.class);
  }

  /**
   * Updates a cursor with a new `after`. Cursors can only be updated forwards
   * (i.e., a cursor cannot be updated with a value that is previous to its
   * current value).
   *
   * @param ctx context object that makes requests to core
   * @param after an indicator of the last transaction processed
   * @return a cursor object
   * @throws ChainException
   */
  public Cursor update(Context ctx, String after) throws ChainException {
    Map<String, Object> req = new HashMap<>();
    req.put("id", this.id);
    req.put("previous_after", this.after);
    req.put("after", after);
    return ctx.request("update-cursor", req, Cursor.class);
  }
}
