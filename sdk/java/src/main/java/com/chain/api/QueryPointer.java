package com.chain.api;

import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;

public class QueryPointer {
    public String index;
    public String chql;
    @SerializedName("chql_params")
    public List<String> params;
    public String cursor;
    @SerializedName("start_time")
    public long startTime;
    @SerializedName("end_time")
    public long endTime;
    public long timestamp;

    public QueryPointer() {
        this.params = new ArrayList<>();
    }
}