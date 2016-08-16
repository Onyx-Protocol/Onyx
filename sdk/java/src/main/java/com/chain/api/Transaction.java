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

    public static class Items extends PagedItems<Transaction> {
        public Items getPage() throws ChainException {
            Items items = this.context.request("list-transactions", this.query, Items.class);
            items.setContext(this.context);
            return items;
        }
    }

    public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
        public Items execute(Context ctx) throws ChainException {
            Items items = new Items();
            items.setContext(ctx);
            items.setQuery(this.query);
            return items.getPage();
        }

        public QueryBuilder setStartTime(long time) {
            this.query.startTime = time;
            return this;
        }

        public QueryBuilder setEndTime(long time) {
            this.query.endTime = time;
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