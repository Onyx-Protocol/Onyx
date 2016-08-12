package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;

import java.math.BigInteger;
import java.util.Map;

public class UnspentOutput {
    @SerializedName("transaction_id")
    public String transactionId;
    public int position;
    @SerializedName("asset_id")
    public String assetId;
    @SerializedName("asset_tags")
    public Map<String, Object> assetTags;
    public BigInteger amount;
    @SerializedName("account_id")
    public String accountId;
    @SerializedName("account_tags")
    public Map<String, Object> accountTags;
    @SerializedName("control_program")
    public byte[] controlProgram;
    @SerializedName("reference_data")
    public Map<String, Object> referenceData;

    public static class Page extends BasePage<UnspentOutput> {

        public Page next(Context ctx)
        throws ChainException {
            return ctx.request("list-unspent-outputs", this.query, Page.class);
        }
    }

    public static class QueryBuilder extends BaseQueryBuilder<QueryBuilder> {
        public Page execute(Context ctx)
        throws ChainException {
            return ctx.request("list-unspent-outputs", this.query, Page.class);
        }

        public QueryBuilder setTimestamp(long time) {
            this.query.timestamp = time;
            return this;
        }
    }
}