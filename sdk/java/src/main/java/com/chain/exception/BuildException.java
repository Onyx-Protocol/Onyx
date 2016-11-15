package com.chain.exception;

import java.util.List;
import com.google.gson.annotations.SerializedName;

/**
 * BuildException wraps errors returned by the build-transaction endpoint.
 */
public class BuildException extends APIException {

  public BuildException(String message, String requestId) {
    super(message, requestId);
  }

  public static class ActionError extends APIException {

    public static class Data {
      /**
       * The index of the action that caused this error.
       */
      @SerializedName("index")
      public Integer index;
    }

    public ActionError(String message, String requestId) {
      super(message, requestId);
    }

    /**
     * Additional data pertaining to the error.
     */
    public Data data;
  }

  public static class Data {
    /**
     * A list of errors resulting from building actions.
     */
    @SerializedName("actions")
    public List<ActionError> actionErrors;
  }

  /**
   * Extra data associated with this error, if any.
   */
  @SerializedName("data")
  public Data data;
}
