package com.chain.api;

import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ControlProgram {
    public byte[] program;

    public static byte[] retireProgram() {
        return "6a".getBytes();
    }

    public static class Builder {
        public String type;
        public Map<String,Object> parameters;

        public Builder() {
            this.parameters = new HashMap<>();
        }

        public ControlProgram create(Context ctx)
        throws ChainException {
            List<ControlProgram> programs = ControlProgram.Builder.create(ctx, Arrays.asList(this));
            return programs.get(0);
        }

        public static List<ControlProgram> create(Context ctx, List<Builder> programs)
        throws ChainException {
            Type type = new TypeToken<List<ControlProgram>>() {}.getType();
            return ctx.request("create-control-program", programs, type);
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

        public Builder addParameter(String key, Object value) {
            this.parameters.put(key, value);
            return this;
        }

        public Builder setParameters(Map<String,Object> parameters) {
            this.parameters = parameters;
            return this;
        }
    }
}