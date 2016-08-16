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

    public static class Items extends PagedItems<UnspentOutput> {
        public Items getPage() throws ChainException {
            Items items = this.context.request("list-unspent-outputs", this.query, Items.class);
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

        public QueryBuilder setTimestamp(long time) {
            this.query.timestamp = time;
            return this;
        }
    }
}