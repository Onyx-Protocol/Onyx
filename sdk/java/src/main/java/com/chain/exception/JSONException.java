package com.chain.exception;

/**
 * JSONException wraps errors due to marshaling/unmarshaling json payloads.
 */
public class JSONException extends ChainException {

  /**
   * Unique indentifier of the request to the server.
   */
  public String requestId;

  /**
   * Initializes exception with its message and requestId attributes.
   * @param message error message
   * @param requestId unique identifier of the request
   */
  public JSONException(String message, String requestId) {
    super(message);
    this.requestId = requestId;
  }

  public String getMessage() {
    return "Message: " + super.getMessage() + " Request-ID: " + this.requestId;
  }
}
