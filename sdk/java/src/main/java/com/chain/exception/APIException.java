package com.chain.exception;

import com.google.gson.annotations.SerializedName;

/**
 * APIException wraps errors returned by the API.
 * Each error contains a brief description in addition to a unique error code.<br>
 * The error code can be used by Chain Support to diagnose the exact cause of the error.
 * The mapping of error codes to messages is as follows:<br><br>
 * <h2>Transaction errors</h2>
 * CH700 - Reference data does not match previous transaction's reference data<br>
 * CH701 - Invalid action type<br>
 * CH702 - Invalid alias on action<br>
 * CH730 - Missing raw transaction<br>
 * CH731 - Too many signing instructions in template for transaction<br>
 * CH732 - Invalid transaction input index<br>
 * CH733 - Invalid signature script component<br>
 * CH734 - Missing signature in template<br>
 * CH735 - Transaction rejected<br>
 * CH760 - Insufficient funds for tx<br>
 * CH761 - Some outputs are reserved; try again<br>
 */
public class APIException extends ChainException {
  public APIException(String message, String requestID) {
    super(message);
    this.requestID = requestID;
  }

  public APIException(String code, String message, String detail, String requestID) {
    super(message);
    this.chainMessage = message;
    this.code = code;
    this.detail = detail;
    this.requestID = requestID;
  }

  public APIException(
      String code, String message, String detail, String requestID, int statusCode) {
    super(message);
    this.chainMessage = message;
    this.code = code;
    this.detail = detail;
    this.requestID = requestID;
    this.statusCode = statusCode;
  }

  @SerializedName("message")
  public String chainMessage;

  public String code;
  public String detail;
  public String requestID;
  public int statusCode;

  public String getMessage() {
    String s = "";

    if (this.code != null && this.code.length() > 0) {
      s += "Code: " + this.code + " ";
    }

    s += "Message: " + this.chainMessage;

    if (this.detail != null && this.detail.length() > 0) {
      s += " Detail: " + this.detail;
    }

    if (this.requestID != null) {
      s += " Request-ID: " + this.requestID;
    }

    return s;
  }
}
