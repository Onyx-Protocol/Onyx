package com.chain.api;

import com.chain.exception.*;
import com.chain.http.*;
import com.google.gson.annotations.SerializedName;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * A predicate to be satisfied when transferring assets.
 */
public class ControlProgram {
  /**
   * Hex-encoded string representation of the control program.
   */
  @SerializedName("control_program")
  public String controlProgram;

  /**
   * Generates hex representation of a "retire" control program.
   * @return hex-encoded "retire" program
   */
  public static String retireProgram() {
    return "6a";
  }

  /**
   * Creates a batch of control programs.
   * @param client client object which makes requests to core
   * @param programs list of control program builder objects
   * @return a list of control programs
   * @throws APIException This exception is raised if the api returns errors while creating the control programs.
   * @throws BadURLException This exception wraps java.net.MalformedURLException.
   * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
   * @throws HTTPException This exception is raised when errors occur making http requests.
   * @throws JSONException This exception is raised due to malformed json requests or responses.
   */
  public static BatchResponse<ControlProgram> createBatch(Client client, List<Builder> programs)
      throws ChainException {
    return client.batchRequest(
        "create-control-program", programs, ControlProgram.class, APIException.class);
  }

  /**
   * ControlProgram.Builder utilizes the builder pattern to create {@link ControlProgram} objects.<br>
   * <strong>If creating an account control program, either {@link #controlWithAccountById(String)}
   * or {@link #controlWithAccountByAlias(String)} must be called before {@link #create(Client)}.</strong>
   */
  public static class Builder {
    /**
     * Specifies the type of control program.
     */
    public String type;

    /**
     * Parameters to the control program.
     */
    public Map<String, Object> params;

    /**
     * Default constructor, initializes the params object.
     */
    public Builder() {
      this.params = new HashMap<>();
    }

    /**
     * Creates a control program.
     * @param client client object which makes requests to core
     * @return a control program
     * @throws APIException This exception is raised if the api returns errors while creating the control program.
     * @throws BadURLException This exception wraps java.net.MalformedURLException.
     * @throws ConnectivityException This exception is raised if there are connectivity issues with the server.
     * @throws HTTPException This exception is raised when errors occur making http requests.
     * @throws JSONException This exception is raised due to malformed json requests or responses.
     */
    public ControlProgram create(Client client) throws ChainException {
      return client.singletonBatchRequest(
          "create-control-program", Arrays.asList(this), ControlProgram.class, APIException.class);
    }

    /**
     * Specifies an account to link to the control program.<br>
     * An id is used to distinguish the account.<br>
     * <strong>If creating an account control program, this or {@link #controlWithAccountByAlias(String)} must be called before {@link #create(Client)}.</strong>
     * @param accountId id of the account
     * @return updated builder object
     */
    public Builder controlWithAccountById(String accountId) {
      this.setType("account");
      this.addParameter("account_id", accountId);
      return this;
    }

    /**
     * Specifies an account to link to the control program.<br>
     * An alias is used to distinguish the account.<br>
     * <strong>If creating an account control program, this or {@link #controlWithAccountById(String)} must be called before {@link #create(Client)}.</strong>
     * @param accountAlias alias of the account
     * @return updated builder object
     */
    public Builder controlWithAccountByAlias(String accountAlias) {
      this.setType("account");
      this.addParameter("account_alias", accountAlias);
      return this;
    }

    /**
     * Sets the type attribute of the control program.
     * @param type the type of control program
     * @return updated builder object
     */
    public Builder setType(String type) {
      this.type = type;
      return this;
    }

    /**
     * Adds a parameter to the control program.
     * @param key the key of the param field
     * @param value the value of the param field
     * @return updated builder object
     */
    public Builder addParameter(String key, Object value) {
      this.params.put(key, value);
      return this;
    }

    /**
     * Sets the parameters object.<br>
     * <strong>Note:</strong> any existing parameter fields will be replaced.
     * @param parameters parameters object
     * @return updated builder object
     */
    public Builder setParameters(Map<String, Object> parameters) {
      this.params = parameters;
      return this;
    }
  }
}
