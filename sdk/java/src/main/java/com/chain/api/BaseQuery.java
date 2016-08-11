package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;

public abstract class BaseQuery<T extends BaseQuery<T>> {
    @SerializedName("query")
    protected QueryPointer queryPointer;

    public abstract <S extends BasePage> S search(Context ctx)
    throws ChainException;

    public BaseQuery() {
      this.queryPointer = new QueryPointer();
    }

    public T useIndex(String index) {
        this.queryPointer.index = index;
        return (T)this;
    }

    public T setQuery(String query) {
        this.queryPointer.query = query;
        return (T)this;
    }

    public T addParameter(String param) {
        this.queryPointer.params.add(param);
        return (T)this;
    }

    public T setParameters(ArrayList<String> params) {
        this.queryPointer.params = new ArrayList<>();
        for (String param : params) {
            this.queryPointer.params.add(param);
        }
        return (T)this;
    }
}
