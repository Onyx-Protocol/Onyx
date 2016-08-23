package com.chain.exception;

public class SigningException extends ChainException {
    private Exception original;

    public SigningException(Exception original) {
        super(original.getMessage());
        this.original = original;
    }

    public SigningException(String message) {
        super(message);
    }

    public Exception getOriginalException() {
        return this.original;
    }
}
