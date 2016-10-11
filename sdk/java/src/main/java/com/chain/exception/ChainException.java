package com.chain.exception;

/**
 * Base exception class for the Chain Core SDK.
 */
public class ChainException extends Exception {
  /**
   * Default constructor.
   */
  public ChainException() {
    super();
  }

  /**
   * Initializes exception with its message attribute.
   * @param message error message
   */
  public ChainException(String message) {
    super(message);
  }
}
