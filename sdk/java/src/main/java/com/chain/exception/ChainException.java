package com.chain.exception;

/**
 * Base exception class for the Chain Core SDK.
 */
public class ChainException extends Exception {
    public ChainException() { super(); }
    public ChainException(String message) { super(message); }
}
