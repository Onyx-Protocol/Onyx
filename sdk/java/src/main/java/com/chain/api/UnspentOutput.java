package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;

import com.google.gson.annotations.SerializedName;

import java.math.BigInteger;
import java.util.List;

public class UnspentOutput {
    @SerializedName("asset_id")
    public String assetId;
    @SerializedName("spent_output")
    public OutputPointer pointer;
    @SerializedName("asset_id")
    public BigInteger amount;
    @SerializedName("asset_tags")
    public List<String> tags;
    @SerializedName("control_program")
    public byte[] controlProgram;
    @SerializedName("reference_data")
    public byte[] referenceData;

    public static class Page extends BasePage<UnspentOutput> {
        public QueryPointerWithTime queryPointer;

        public Page next(Context ctx)
                throws ChainException {
            return ctx.request("list-unspent-outputs", this.queryPointer, Page.class);
        }
    }

    public static class Query extends BaseQuery<Page> {
        public QueryPointerWithTime queryPointer;

        public Page search(Context ctx)
                throws ChainException {
            return ctx.request("list-unspent-outputs", this.queryPointer, Page.class);
        }

        public BigInteger sum(Context ctx)
                throws ChainException {
            return ctx.request("sum-unspent-outputs", this.queryPointer, Page.class);
        }

        public Query setStartTime(long time) {
            this.queryPointer.startTime = time;
            return this;
        }

        public Query setEndTime(long time) {
            this.queryPointer.endTime = time;
            return this;
        }
    }

    static class QueryPointerWithTime extends QueryPointer {
        public long startTime;
        public long endTime;
    }
}