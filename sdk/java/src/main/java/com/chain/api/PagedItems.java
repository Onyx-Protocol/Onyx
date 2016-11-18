package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Client;
import com.google.gson.annotations.Expose;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

/**
 * Abstract base class representing api query results.
 * @param <T> type of api object
 */
public abstract class PagedItems<T> implements Iterator<T> {
  /**
   * Client object that makes the query requests.
   */
  protected Client client;

  /**
   * Pointer to the current item in the results list.
   */
  private int pos;

  /**
   * Page of api objects returned from the most recent query.
   */
  @Expose(serialize = false)
  @SerializedName("items")
  public List<T> list;

  /**
   * Specifies if the current page of results is the last.
   */
  @Expose(serialize = false)
  @SerializedName("last_page")
  public boolean lastPage;

  /**
   * Specifies the details of the next query.
   */
  public Query next;

  /**
   * Retrieves the next page of results.
   * @return a paged collection of type T
   * @throws ChainException
   */
  public abstract PagedItems<T> getPage() throws ChainException;

  public PagedItems() {
    this.pos = 0;
    this.list = new ArrayList<>();
    this.lastPage = false;
  }

  /**
   * Sets the client object.
   * @param client context object which makes requests to core
   */
  public void setClient(Client client) {
    this.client = client;
  }

  /**
   * Sets the next query object.
   * @param next query object for the next request
   */
  public void setNext(Query next) {
    this.next = next;
  }

  /**
   * Returns the next item in the results list.
   * @return api object of type T
   */
  public T next() {
    return list.get(pos++);
  }

  /**
   * Returns true if there is another item in the results list.
   * @return boolean
   */
  public boolean hasNext() {
    if (pos < list.size()) {
      return true;
    } else {
      if (!lastPage) {
        try {
          PagedItems<T> items = this.getPage();
          this.pos = 0;
          this.list = items.list;
          this.lastPage = items.lastPage;
          this.next = items.next;

          return this.list.size() > 0;
        } catch (ChainException e) {
          return false;
        }
      } else {
        return false;
      }
    }
  }

  /**
   * This method is unsupported.
   * @throws UnsupportedOperationException
   */
  public void remove() throws UnsupportedOperationException {
    throw new UnsupportedOperationException();
  }
}
