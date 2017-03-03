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
   * Initializes a new exception while storing the original cause.
   * @param cause the cause of the exception
   * @param message error message
   */
  public ChainException(String message, Throwable cause) {
    super(message, cause);
  }

  /**
   * Initializes exception with its message attribute.
   * @param message error message
   */
  public ChainException(String message) {
    super(message);
  }
}
