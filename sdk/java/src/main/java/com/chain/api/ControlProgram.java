package com.chain.api;

import com.chain.exception.APIException;
import com.chain.exception.ChainException;
import com.chain.http.Context;
import com.google.gson.annotations.SerializedName;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ControlProgram {
  @SerializedName("control_program")
  public String program;

  // Error data
  public String code;
  public String message;
  public String detail;

  public static String retireProgram() {
    return "6a";
  }

  public static class Builder {
    public String type;
    public Map<String, Object> parameters;

    public Builder() {
      this.parameters = new HashMap<>();
    }

    public ControlProgram create(Context ctx) throws ChainException {
      List<ControlProgram> programs = ControlProgram.Builder.create(ctx, Arrays.asList(this));
      ControlProgram result = programs.get(0);
      if (result.program == null || result.program.isEmpty()) {
        throw new APIException(result.code, result.message, result.detail, null);
      }
      return result;
    }

    public static List<ControlProgram> create(Context ctx, List<Builder> programs)
        throws ChainException {
      Type type = new TypeToken<List<ControlProgram>>() {}.getType();
      return ctx.request("create-control-program", programs, type);
    }

    public Builder controlWithAccountById(String accountId) {
      this.type = "account";
      this.addParameter("account_id", accountId);
      return this;
    }

    public Builder controlWithAccountByAlias(String accountAlias) {
      this.type = "account";
      this.addParameter("account_alias", accountAlias);
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

    public Builder setParameters(Map<String, Object> parameters) {
      this.parameters = parameters;
      return this;
    }
  }
}
