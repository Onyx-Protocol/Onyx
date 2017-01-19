package com.chain.exception;

import com.chain.proto.Error;
import com.google.gson.annotations.SerializedName;

/**
 * APIException wraps errors returned by the API.
 * Each error contains a brief description in addition to a unique error code.<br>
 * The error code can be used by Chain Support to diagnose the exact cause of the error.
 * The mapping of error codes to messages is as follows:<br><br>
 *
 * <h2>General errors</h2>
 * CH001 - Request timed out
 * CH002 - Not found
 * CH003 - Invalid request body
 * CH004 - Invalid request header
 * CH006 - Not found
 *
 * <h2>Account/Asset errors</h2>
 * CH200 - Quorum must be greater than 1 and less than or equal to the length of xpubs
 * CH201 - Invalid xpub format
 * CH202 - At least one xpub is required
 * CH203 - Retrieved type does not match expected type
 *
 * <h2>Access token errors</h2>
 * CH300 - Malformed or empty access token id
 * CH301 - Access tokens must be type client or network
 * CH302 - Access token id is already in use
 * CH310 - The access token used to authenticate this request cannot be deleted
 *
 * <h2>Query errors</h2>
 * CH600 - Malformed pagination parameter `after`
 * CH601 - Incorrect number of parameters to filter
 * CH602 - Malformed query filter
 *
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
  /**
   * Unique identifier for the error.
   */
  public String code;

  /**
   * Message describing the general nature of the error.
   */
  @SerializedName("message")
  public String chainMessage;

  /**
   * Additional information about the error (possibly null).
   */
  public String detail;

  /**
   * Specifies whether the error is temporary, or a change to the request is necessary.
   */
  public boolean temporary;

  /**
   * Unique identifier of the request to the server.
   */
  public String requestId;

  /**
   * HTTP status code returned by the server.
   */
  public int statusCode;

  /**
   * Initializes exception with its message and requestId attributes.
   * @param message error message
   * @param requestId unique identifier of the request
   */
  public APIException(String message, String requestId) {
    super(message);
    this.requestId = requestId;
  }

  /**
   * Intitializes exception with its code, message, detail &amp; temporary field set.
   * @param code error code
   * @param message error message
   * @param detail additional error information
   * @param temporary unique identifier of the request
   */
  public APIException(String code, String message, String detail, boolean temporary) {
    super(message);
    this.chainMessage = message;
    this.code = code;
    this.detail = detail;
    this.temporary = temporary;
  }

  /**
   * Initializes exception with its code, message, detail &amp; requestId attributes.
   * @param code error code
   * @param message error message
   * @param detail additional error information
   * @param requestId unique identifier of the request
   */
  public APIException(String code, String message, String detail, String requestId) {
    super(message);
    this.chainMessage = message;
    this.code = code;
    this.detail = detail;
    this.requestId = requestId;
  }

  public APIException(Error error) {
    super(error.getMessage());
    this.chainMessage = error.getMessage();
    this.code = error.getCode();
    this.detail = error.getDetail();
    this.temporary = error.getTemporary();
  }

  /**
   * Initializes exception with all of its attributes.
   * @param code error code
   * @param message error message
   * @param detail additional error information
   * @param temporary specifies if the error is temporary
   * @param requestId unique identifier of the request
   * @param statusCode HTTP status code
   */
  public APIException(
      String code,
      String message,
      String detail,
      boolean temporary,
      String requestId,
      int statusCode) {
    super(message);
    this.chainMessage = message;
    this.code = code;
    this.detail = detail;
    this.temporary = temporary;
    this.requestId = requestId;
    this.statusCode = statusCode;
  }

  /**
   * Constructs a detailed message of the error.
   * @return detailed error message
   */
  @Override
  public String getMessage() {
    String s = "";

    if (this.code != null && this.code.length() > 0) {
      s += "Code: " + this.code + " ";
    }

    s += "Message: " + this.chainMessage;

    if (this.detail != null && this.detail.length() > 0) {
      s += " Detail: " + this.detail;
    }

    if (this.requestId != null) {
      s += " Request-ID: " + this.requestId;
    }

    return s;
  }
}
