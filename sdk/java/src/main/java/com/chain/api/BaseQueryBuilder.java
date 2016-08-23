package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;

public abstract class BaseQueryBuilder<T extends BaseQueryBuilder<T>> {
    @SerializedName("query")
    protected Query query;

    public abstract <S extends PagedItems> S execute(Context ctx)
    throws ChainException;

    public BaseQueryBuilder() {
      this.query = new Query();
    }

    public T useIndexById(String id) {
        this.query.indexId = id;
        return (T)this;
    }

    public T useIndexByAlias(String alias) {
        this.query.indexAlias = alias;
        return (T)this;
    }

    public T withChQL(String chql) {
        this.query.chql = chql;
        return (T)this;
    }

    public T addChQLParameter(String param) {
        this.query.chqlParams.add(param);
        return (T)this;
    }

    public T setChQLParameters(ArrayList<String> params) {
        this.query.chqlParams = new ArrayList<>();
        for (String cp : params) {
            this.query.chqlParams.add(cp);
        }
        return (T)this;
    }
}
