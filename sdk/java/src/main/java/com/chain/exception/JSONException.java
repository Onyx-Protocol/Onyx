package com.chain.exception;

public class JSONException extends ChainException {
  public JSONException(String message, String requestID) {
    super(message);
    this.requestID = requestID;
  }

  public String requestID;

  public String getMessage() {
    return "Message: " + super.getMessage() + " Request-ID: " + this.requestID;
  }
}
