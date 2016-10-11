package com.chain.exception;

/**
 * BadURLException wraps errors due to malformed URLs.
 */
public class BadURLException extends ChainException {
  /**
   * Initializes exception with its message attribute.
   * @param message error message
   */
  public BadURLException(String message) {
    super(message);
  }
}
