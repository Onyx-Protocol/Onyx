package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;

import java.util.HashMap;
import java.util.Map;

public class ControlProgram {
    public byte[] program;

    public static class Builder {
        public String type;
        public Map<String,Object> parameters;

        public Builder() {
            this.parameters = new HashMap<>();
        }

        public ControlProgram create(Context ctx)
                throws ChainException {
            return ctx.request("create-control-program", this, ControlProgram.class);
        }

        public Builder controlWithAccount(String accountId) {
            this.type = "account";
            this.addParameter("account_id", accountId);
            return this;
        }

        public Builder setType(String type) {
            this.type = type;
            return this;
        }

        public Builder setParameters(Map<String,Object> parameters) {
            this.parameters = parameters;
            return this;
        }

        public Builder addParameter(String key, Object value) {
            this.parameters.put(key, value);
            return this;
        }
    }
}
