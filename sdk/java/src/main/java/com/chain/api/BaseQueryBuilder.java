package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Client;

import java.util.ArrayList;
import java.util.List;

/**
 * Abstract base class providing interface for building queries on API objects.
 * @param <T> the QueryBuilder class that extends BaseQueryBuilder
 */
public abstract class BaseQueryBuilder<T extends BaseQueryBuilder<T>> {
  /**
   * The query information that will be used when the next page of results is requested from the server.
   */
  protected Query next;

  /**
   * Executes the api query.
   * @param client context object which makes requests to core
   * @return a page of S objects
   * @throws ChainException
   */
  public abstract <S extends PagedItems> S execute(Client client) throws ChainException;

  /**
   * Default constructor initializes the next query.
   */
  public BaseQueryBuilder() {
    this.next = new Query();
  }

  /**
   * Sets the after attribute on the query builder object.
   * @param after specifies where the last item returned from the current query
   * @return updated builder object
   */
  public T setAfter(String after) {
    this.next.after = after;
    return (T) this;
  }

  /**
   * Sets the filter attribute on the query builder object.
   * @param filter the predicate used to filter results
   * @return updated builder object
   */
  public T setFilter(String filter) {
    this.next.filter = filter;
    return (T) this;
  }

  /**
   * Adds filter parameters to the query builder object.
   * @param param parameter to be added
   * @return updated builder object
   */
  public T addFilterParameter(Object param) {
    this.next.filterParams.add(param);
    return (T) this;
  }

  /**
   * Sets the filter parameters list.<br>
   * <strong>Note:</strong> any existing filter params will be replaced.
   * @param params list of parameters to be added
   */
  public T setFilterParameters(List<?> params) {
    this.next.filterParams = new ArrayList<>(params);
    return (T) this;
  }
}
