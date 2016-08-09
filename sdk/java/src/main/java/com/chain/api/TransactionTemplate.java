package com.chain.api;

import com.chain.http.Context;

import java.lang.reflect.Type;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;

import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;
import com.chain.exception.ChainException;

public class TransactionTemplate {

    @SerializedName("unsigned_hex")
    public String unsignedHex;
    @SerializedName("block_chain")
    public String blockChain;
    public List<Input> inputs;

    public static class Input {
        @SerializedName("asset_id")
        public String assetID;
        public BigInteger amount;
        @SerializedName("signature_components")
        public SignatureComponent[] signatureComponents;
    }

    public static class SignatureComponent {
        public String type;
        public String data;
        public int required;
        @SerializedName("signature_data")
        public String signatureData;
        public Signature[] signatures;
    }

    public static class Signature {
        public String xpub;
        @SerializedName("derivation_path")
        public ArrayList<Integer> derivationPath;
        public String signature;
    }

    public static List<TransactionTemplate> build(Context ctx, List<TransactionTemplate.Builder> templates)
            throws ChainException {
        Type type = new TypeToken<ArrayList<TransactionTemplate>>() {}.getType();
        return ctx.request("build-transaction-template", templates, type);
    }

    public static class Action {
        public String type;
        public HashMap<String, Object> params;
        public byte[] referenceData;

        public Action() {
            this.params = new HashMap();
        }

        public Action setType(String type) {
            this.type = type;
            return this;
        }

        public Action setParameter(String key, Object value) {
            this.params.put(key, value);
            return this;
        }

        public Action setReferenceData(byte[] referenceData) {
            this.referenceData = referenceData;
            return this;
        }
    }


    public static class Builder {
        private List<Action> actions;
        private byte[] referenceData;

        public TransactionTemplate build(Context ctx)
                throws ChainException {
            List<TransactionTemplate.Builder> req = new ArrayList<TransactionTemplate.Builder>();
            req.add(this);

            Type type = new TypeToken<ArrayList<TransactionTemplate>>() {}.getType();
            List<TransactionTemplate> tmpls = ctx.request("build-transaction-template", req, type);
            return tmpls.get(0);
        }

        public Builder() {
            this.actions = new ArrayList<>();
        }

        public Builder addAction(Action action) {
            this.actions.add(action);
            return this;
        }

        public Builder setReferenceData(byte[] referenceData) {
            this.referenceData = referenceData;
            return this;
        }
    }
}