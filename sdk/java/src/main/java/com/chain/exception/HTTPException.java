package com.chain.exception;

/**
 * HTTPException wraps generic HTTP errors.
 */
public class HTTPException extends ChainException {
  /**
   * Initializes exception with its message attribute.
   * @param message error message
   */
  public HTTPException(String message) {
    super(message);
  }

  /**
   * Initializes exception while storing original cause
   */
  public HTTPException(String message, Throwable cause) {
    super(message, cause);
  }
}
