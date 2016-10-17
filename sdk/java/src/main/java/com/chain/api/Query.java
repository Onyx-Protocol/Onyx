package com.chain.api;

import com.google.gson.annotations.SerializedName;

import java.util.ArrayList;
import java.util.List;

public class Query {
  public String filter;

  @SerializedName("filter_params")
  public List<String> filterParams;

  @SerializedName("ascending_with_long_poll")
  public boolean ascendingWithLongPoll;

  public long timeout;

  public String after;

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
