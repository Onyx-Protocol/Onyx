package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;

import com.google.gson.annotations.SerializedName;

import java.math.BigInteger;

import java.util.List;
import java.util.Map;

public class Transaction {
    @SerializedName("block_height")
    public int blockHeight;
    @SerializedName("block_id")
    public String blockId;
    public String id;
    public List<Input> inputs;
    public List<Output> outputs;
    public int position;
    @SerializedName("reference_data")
    public Map<String, Object> referenceData;

    public static class Page extends BasePage<Transaction> {
        public Page next(Context ctx)
        throws ChainException {
            return ctx.request("list-transactions", this.queryPointer, Page.class);
        }
    }

    public static class Query extends BaseQuery<Query> {
        public Query() {
            this.queryPointer = new QueryPointer();
        }

        public Page search(Context ctx)
        throws ChainException {
            return ctx.request("list-transactions", this.queryPointer, Page.class);
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

    public static class Input {
        public String action;
        public BigInteger amount;
        @SerializedName("asset_id")
        public String assetId;
        @SerializedName("account_id")
        public String accountId;
        @SerializedName("account_tags")
        public Map<String, Object> accountTags;
        @SerializedName("asset_tags")
        public Map<String, Object> assetTags;
        @SerializedName("input_witness")
        public byte[][] inputWitness;
        @SerializedName("issuance_program")
        public byte[] issuanceProgram;
        @SerializedName("reference_data")
        public Map<String, Object> referenceData;
    }

    public static class Output {
        public String action;
        public BigInteger amount;
        @SerializedName("asset_id")
        public String assetId;
        @SerializedName("control_program")
        public byte[] controlProgram;
        public int position;
        @SerializedName("account_id")
        public String accountId;
        @SerializedName("account_tags")
        public Map<String, Object> accountTags;
        @SerializedName("asset_tags")
        public Map<String, Object> assetTags;
        @SerializedName("reference_data")
        public Map<String, Object> referenceData;
    }
}