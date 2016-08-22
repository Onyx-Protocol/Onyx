package com.chain;

import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.math.BigInteger;
import java.util.*;

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

    public static class Template {
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
    }

    public static class SubmitResponse {
        public String id;

        // Error data
        public String code;
        public String message;
        public String detail;
    }

    public static List<Template> build(Context ctx, List<Transaction.Builder> templates) throws ChainException {
        Type type = new TypeToken<ArrayList<Template>>() {
        }.getType();
        return ctx.request("build-transaction-template", templates, type);
    }

    public static List<SubmitResponse> submit(Context ctx, List<Template> templates) throws ChainException {
        Type type = new TypeToken<ArrayList<SubmitResponse>>() {
        }.getType();

        HashMap<String, Object> requestBody = new HashMap<>();
        requestBody.put("transactions", templates);

        return ctx.request("submit-transaction-template", requestBody, type);
    }

    public static class Action {
        public String type;
        public HashMap<String, Object> params;
        @SerializedName("reference_data")
        public Map<String, Object> referenceData;
        @SerializedName("client_token")
        private String clientToken;

        public Action() {
            this.params = new HashMap();
            this.clientToken = UUID.randomUUID().toString();
        }

        public Action setType(String type) {
            this.type = type;
            return this;
        }

        public Action setParameter(String key, Object value) {
            this.params.put(key, value);
            return this;
        }

        public Action setReferenceData(Map<String, Object> referenceData) {
            this.referenceData = referenceData;
            return this;
        }
    }


    public static class Builder {
        private List<Action> actions;
        @SerializedName("reference_data")
        private Map<String, Object> referenceData;

        public Template build(Context ctx) throws ChainException {
            List<Template> tmpls = Transaction.build(ctx, Arrays.asList(this));
            return tmpls.get(0);
        }

        public Builder() {
            this.actions = new ArrayList<>();
        }

        public Builder addAction(Action action) {
            this.actions.add(action);
            return this;
        }

        public Builder addAction(Action action, Map<String, Object> referenceData) {
            if (referenceData != null) {
                action.setReferenceData(referenceData);
            }

            this.actions.add(action);
            return this;
        }

        public Builder setReferenceData(Map<String, Object> referenceData) {
            this.referenceData = referenceData;
            return this;
        }

        public Builder issueById(String assetId, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("issue")
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder issueByAlias(String assetAlias, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("issue")
                    .setParameter("asset_alias", assetAlias)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder controlWithAccountByID(String accountId, String assetId, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("control_account")
                    .setParameter("account_id", accountId)
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder controlWithAccountByAlias(String accountAlias, String assetAlias, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("control_account")
                    .setParameter("account_alias", accountAlias)
                    .setParameter("asset_alias", assetAlias)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder controlWithProgramById(ControlProgram program, String assetId, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("control_program")
                    .setParameter("control_program", program.program)
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder controlWithProgramByAlias(ControlProgram program, String assetAlias, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("control_program")
                    .setParameter("control_program", program.program)
                    .setParameter("asset_alias", assetAlias)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder spendFromAccountById(String accountId, String assetId, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("spend_account_unspent_output_selector")
                    .setParameter("account_id", accountId)
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder spendFromAccountByAlias(String accountAlias, String assetAlias, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("spend_account_unspent_output_selector")
                    .setParameter("account_alias", accountAlias)
                    .setParameter("asset_alias", assetAlias)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder spendUnspentOutput(UnspentOutput uo, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("spend_account_unspent_output")
                    .setParameter("transaction_id", uo.transactionId)
                    .setParameter("position", uo.position);

            return this.addAction(action, referenceData);
        }

        public Builder spendUnspentOutputs(List<UnspentOutput> uos, Map<String, Object> referenceData) {
            for (UnspentOutput uo : uos) {
                this.spendUnspentOutput(uo, referenceData);
            }

            return this;
        }

        public Builder retireById(String assetId, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("control_program")
                    .setParameter("control_program", ControlProgram.retireProgram())
                    .setParameter("asset_id", assetId)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }

        public Builder retireByAlias(String assetAlias, BigInteger amount, Map<String, Object> referenceData) {
            Action action = new Action()
                    .setType("control_program")
                    .setParameter("control_program", ControlProgram.retireProgram())
                    .setParameter("asset_alias", assetAlias)
                    .setParameter("amount", amount);

            return this.addAction(action, referenceData);
        }
    }
}
