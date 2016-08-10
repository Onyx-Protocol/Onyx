package com.chain.api;

import com.google.gson.annotations.SerializedName;

public class OutputPointer {
    @SerializedName("transaction_id")
    public String transactionId;
    public int index;
}