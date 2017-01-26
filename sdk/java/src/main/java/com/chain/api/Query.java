package com.chain.api;

import com.google.gson.annotations.SerializedName;
import com.google.protobuf.ByteString;

import java.util.ArrayList;
import java.util.List;

/**
 * Stores information used to execute item queries on the Chain Core.
 */
public class Query {
  /**
   * Predicate used to filter query results.
   */
  public String filter;

  /**
   * Parameters used in the query's filter.
   */
  @SerializedName("filter_params")
  public List<FilterParam> filterParams;

  /**
   * Specifies if this query is being used within a transaction feed.<br>
   * If true, the query will long poll until the request returns results or times out.
   */
  @SerializedName("ascending_with_long_poll")
  public boolean ascendingWithLongPoll;

  /**
   * Specifies a timeout for transaction feed queries.
   */
  public long timeout;

  /**
   * Represents a bookmark to the last returned item. The next query will return results
   * starting after this item.
   */
  public String after;

  /**
   * Specifies the earliest transaction timestamp (in milliseconds) to include in transaction query results.
   */
  @SerializedName("start_time")
  public long startTime;

  /**
   * Specifies the latest transaction timestamp (in milliseconds) to include in transaction query results.
   */
  @SerializedName("end_time")
  public long endTime;

  /**
   * Specifies a time for point-in-time queries, e.g. balances or unspent outputs.
   */
  public long timestamp;

  /**
   * Specifies parameters to sum by when executing balance queries.
   */
  @SerializedName("sum_by")
  public List<String> sumBy;

  /**
   * Specifies aliases to use when filteringer results. This is parameter only used in {@link MockHsm.Key} queries.
   */
  public List<String> aliases;

  /**
   * Default constructor initializes filter parameters and sum by lists.
   */
  public Query() {
    this.filterParams = new ArrayList<>();
    this.sumBy = new ArrayList<>();
    this.aliases = new ArrayList<>();
  }

  public static abstract class FilterParam {
    public abstract com.chain.proto.FilterParam toProtobuf();

    public static class StringParam extends FilterParam {
      private String value;

      public StringParam(String param) {
        value = param;
      }

      public com.chain.proto.FilterParam toProtobuf() {
        return com.chain.proto.FilterParam.newBuilder().setString(value).build();
      }
    }

    public static class BoolParam extends FilterParam {
      private boolean value;

      public BoolParam(boolean param) {
        value = param;
      }

      public com.chain.proto.FilterParam toProtobuf() {
        return com.chain.proto.FilterParam.newBuilder().setBool(value).build();
      }
    }

    public static class LongParam extends FilterParam {
      private long value;

      public LongParam(long param) {
        value = param;
      }

      public com.chain.proto.FilterParam toProtobuf() {
        return com.chain.proto.FilterParam.newBuilder().setInt64(value).build();
      }
    }

    public static class BytesParam extends FilterParam {
      private byte[] value;

      public BytesParam(byte[] param) {
        value = param;
      }

      public com.chain.proto.FilterParam toProtobuf() {
        return com.chain.proto.FilterParam.newBuilder()
            .setBytes(ByteString.copyFrom(value))
            .build();
      }
    }
  }
}
