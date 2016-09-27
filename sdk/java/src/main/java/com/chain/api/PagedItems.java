package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.Expose;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

public abstract class PagedItems<T> implements Iterator<T> {
  protected Context context;
  private int pos;

  @Expose(serialize = false)
  @SerializedName("items")
  public List<T> list;

  @Expose(serialize = false)
  @SerializedName("last_page")
  public boolean lastPage;

  public Query next;

  public abstract PagedItems<T> getPage() throws ChainException;

  public PagedItems() {
    this.pos = 0;
    this.list = new ArrayList<>();
    this.lastPage = false;
  }

  public void setContext(Context context) {
    this.context = context;
  }

  public void setNext(Query next) {
    this.next = next;
  }

  public T next() {
    return list.get(pos++);
  }

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
}
