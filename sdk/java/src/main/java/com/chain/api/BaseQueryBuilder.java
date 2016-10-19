package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Client;

import java.util.ArrayList;

public abstract class BaseQueryBuilder<T extends BaseQueryBuilder<T>> {
  protected Query next;

  public abstract <S extends PagedItems> S execute(Client client) throws ChainException;

  public BaseQueryBuilder() {
    this.next = new Query();
  }

  public T setAfter(String after) {
    this.next.after = after;
    return (T) this;
  }

  public T setFilter(String filter) {
    this.next.filter = filter;
    return (T) this;
  }

  public T addFilterParameter(String param) {
    this.next.filterParams.add(param);
    return (T) this;
  }

  public T setFilterParameters(ArrayList<String> params) {
    this.next.filterParams = new ArrayList<>(params);
    return (T) this;
  }
}
