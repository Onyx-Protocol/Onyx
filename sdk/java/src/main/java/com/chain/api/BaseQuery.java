package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;

public abstract class BaseQuery<T> {
    @SerializedName("query")
    public QueryPointer queryPointer;

    public abstract T search(Context ctx)
    throws ChainException;

    public BaseQuery() {
      this.queryPointer = new QueryPointer();
    }

    public <S extends BaseQuery> S useIndex(String indexId) {
        this.queryPointer.indexId = indexId;
        return (S)this;
    }

    public <S extends BaseQuery> S addParameter(String parameter) {
        this.queryPointer.parameters.add(parameter);
        return (S)this;
    }

    public <S extends BaseQuery> S setParameters(ArrayList<String> parameters) {
        this.queryPointer.parameters = parameters;
        return (S)this;
    }
}
