package com.chain.api;

import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;

public class Query {
    @SerializedName("index_id")
    public String indexId;
    @SerializedName("index_alias")
    public String indexAlias;
    public String filter;
    @SerializedName("filter_params")
    public List<String> filterParams;
    public String cursor;
    @SerializedName("start_time")
    public long startTime;
    @SerializedName("end_time")
    public long endTime;
    public long timestamp;
    @SerializedName("sum_by")
    public List<String> sumBy;

    public Query() {
        this.filterParams = new ArrayList<>();
        this.sumBy = new ArrayList<>();
    }
}