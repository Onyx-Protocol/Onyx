package com.chain.api;

import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;

public class Query {
    public String index;
    public String chql;
    @SerializedName("chql_params")
    public List<String> chqlParams;
    public String cursor;
    @SerializedName("start_time")
    public long startTime;
    @SerializedName("end_time")
    public long endTime;
    public long timestamp;

    public Query() {
        this.chqlParams = new ArrayList<>();
    }
}