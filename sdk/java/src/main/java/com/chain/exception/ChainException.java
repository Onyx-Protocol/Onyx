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

  /**
   * Initializes a new exception while storing the original cause.
   * @param message the error message
   * @param cause the cause of the exception
   */
  public ChainException(String message, Throwable cause) {
    super(message, cause);
  }
}
