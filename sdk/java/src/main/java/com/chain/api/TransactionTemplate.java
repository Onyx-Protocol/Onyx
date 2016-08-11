package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;

import java.lang.reflect.Type;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;

import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

public class TransactionTemplate {
    @SerializedName("unsigned_hex")
    public String unsignedHex;
    public List<Input> inputs;

    public static class Input {
        @SerializedName("asset_id")
        public String assetID;
        public BigInteger amount;
        public int position;
        @SerializedName("signature_components")
        public SignatureComponent[] signatureComponents;
    }

    public static class SignatureComponent {
        public String type;
        public String data;
        public int quorum;
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

    public static class SubmitResponse {
        public String id;

        // Error data
        public String code;
        public String message;
        public String detail;
    }

    public static List<TransactionTemplate> build(Context ctx, List<TransactionTemplate.Builder> templates)
    throws ChainException {
        Type type = new TypeToken<ArrayList<TransactionTemplate>>() {}.getType();
        return ctx.request("build-transaction-template", templates, type);
    }

    public SubmitResponse submit(Context ctx, TransactionTemplate template)
    throws ChainException {
        List<SubmitResponse> transactions = TransactionTemplate.submit(ctx, Arrays.asList(template));
        return transactions.get(0);
    }

    public static List<SubmitResponse> submit(Context ctx, List<TransactionTemplate> templates)
    throws ChainException {
        Type type = new TypeToken<ArrayList<SubmitResponse>>() {}.getType();

        HashMap<String, Object> requestBody = new HashMap<>();
        requestBody.put("transactions", templates);

        return ctx.request("submit-transaction-template", requestBody, type);
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
            List<TransactionTemplate> tmpls = TransactionTemplate.build(ctx, Arrays.asList(this));
            return tmpls.get(0);
        }

        public Builder() {
            this.actions = new ArrayList<>();
        }

        public Builder addAction(Action action) {
            this.actions.add(action);
            return this;
        }

        public Builder addAction(Action action, byte[] referenceData) {
            if (referenceData != null) {
                action.setReferenceData(referenceData);
            }

            this.actions.add(action);
            return this;
        }

        public Builder setReferenceData(byte[] referenceData) {
            this.referenceData = referenceData;
            return this;
        }

        public Builder issue(String assetId, BigInteger amount, byte[] referenceData) {
            Action action = new Action()
                    .setType("issue")
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

         public Builder controlWithAccount(String accountId, String assetId, BigInteger amount, byte[] referenceData) {
             Action action = new Action()
                     .setType("control_account")
                     .setParameter("account_id", accountId)
                     .setParameter("asset_id", assetId)
                     .setParameter("amount", amount);

             return this.addAction(action, referenceData);
        }

        public Builder controlWithProgram(ControlProgram program, String assetId, BigInteger amount, byte[] referenceData) {
            Action action = new Action()
                    .setType("control_program")
                    .setParameter("control_program", program.program)
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }


        public Builder spendFromAccount(String accountId, String assetId, BigInteger amount, byte[] referenceData) {
            Action action = new Action()
                    .setType("spend_account_unspent_output_selector")
                    .setParameter("account_id", accountId)
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder spendUnspentOutput(UnspentOutput uo, byte[] referenceData) {
            Action action = new Action()
                    .setType("spend_account_unspent_output_selector")
                    .setParameter("transaction_hash", uo.pointer.transactionId)
                    .setParameter("transaction_output", uo.pointer.index)
                    .setParameter("asset_id", uo.assetId)
                    .setParameter("amount", uo.amount);

            return this.addAction(action, referenceData);
        }

        public Builder spendUnspentOutputs(List<UnspentOutput> uos, byte[] referenceData) {
            for (UnspentOutput uo : uos) {
                this.spendUnspentOutput(uo, referenceData);
            }

            return this;
        }

        public Builder retire(String assetId, BigInteger amount, byte[] referenceData) {
            Action action = new Action()
                    .setType("control_program")
                    .setParameter("control_program", ControlProgram.retireProgram())
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }
    }
}
