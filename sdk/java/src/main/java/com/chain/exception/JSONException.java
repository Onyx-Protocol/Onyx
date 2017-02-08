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
   * Default constructor.
   */
  public JSONException(String message) {
    super(message);
  }

  /**
   * Initializes exception with its message and requestId attributes.
   * Use this constructor in context of an API call.
   *
   * @param message error message
   * @param requestId unique identifier of the request
   */
  public JSONException(String message, String requestId) {
    super(message);
    this.requestId = requestId;
  }

  public String getMessage() {
    String message = "Message: " + super.getMessage();
    if (requestId != null) {
      message += " Request-ID: " + requestId;
    }
    return message;
  }
}
