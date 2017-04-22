package com.chain.exception;

/**
 * ConfigurationException wraps errors during client configuration.
 */
public class ConfigurationException extends ChainException {
  /**
   * Initializes exception with its message attribute.
   * @param message error message
   */
  public ConfigurationException(String message) {
    super(message);
  }

  /**
   * Initializes new exception while storing original cause.
   * @param message the error message
   * @param cause the original cause
   */
  public ConfigurationException(String message, Throwable cause) {
    super(message, cause);
  }
}
