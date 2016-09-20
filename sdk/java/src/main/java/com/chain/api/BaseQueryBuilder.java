package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;

public abstract class BaseQueryBuilder<T extends BaseQueryBuilder<T>> {
  @SerializedName("query")
  protected Query query;

  public abstract <S extends PagedItems> S execute(Context ctx) throws ChainException;

  public BaseQueryBuilder() {
    this.query = new Query();
  }

  public T useIndexById(String id) {
    this.query.indexId = id;
    return (T) this;
  }

  public T useIndexByAlias(String alias) {
    this.query.indexAlias = alias;
    return (T) this;
  }

  public T withFilter(String filter) {
    this.query.filter = filter;
    return (T) this;
  }

  public T addFilterParameter(String param) {
    this.query.filterParams.add(param);
    return (T) this;
  }

  public T setFilterParameters(ArrayList<String> params) {
    this.query.filterParams = new ArrayList<>();
    for (String p : params) {
      this.query.filterParams.add(p);
    }
    return (T) this;
  }
}
