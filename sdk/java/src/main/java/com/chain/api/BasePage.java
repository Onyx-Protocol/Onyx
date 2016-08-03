package com.chain.api;

import com.google.gson.annotations.Expose;
import com.google.gson.annotations.SerializedName;

import java.util.List;
import com.chain.http.Context;
import com.chain.exception.ChainException;

public abstract class BasePage<T> {
    @Expose(serialize = false)
    public List<T> items;

    @Expose(serialize = false)
    @SerializedName("last_page")
    public boolean lastPage;

    @SerializedName("query")
    public QueryPointer queryPointer;

    public abstract <S extends BasePage> S next(Context ctx)
    throws ChainException;

    public boolean hasNext() {
        return lastPage;
    }
}
