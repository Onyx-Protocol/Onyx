package com.chain;

import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;

public class Query {
    @SerializedName("index_id")
    public String indexId;
    @SerializedName("index_alias")
    public String indexAlias;
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
