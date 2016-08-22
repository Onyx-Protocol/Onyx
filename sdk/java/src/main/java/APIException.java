package com.chain;

/**
 * APIException wraps errors returned by the API.
 * Each error contains a brief description in addition
 * to a unique error code. The error code can be used by
 * Chain Support to diagnose the exact cause of the error.
 */
public class APIException extends ChainException {
    public APIException(String message, String requestID) {
        super(message);
        this.requestID = requestID;
    }

    public APIException(String code, String message, String detail, String requestID) {
        super(message);
        this.chainMessage = message;
        this.code = code;
        this.detail = detail;
        this.requestID = requestID;
    }

    public APIException(String code, String message, String detail, String requestID, int statusCode) {
        super(message);
        this.chainMessage = message;
        this.code = code;
        this.detail = detail;
        this.requestID = requestID;
        this.statusCode = statusCode;
    }

    public String chainMessage;
    public String code;
    public String detail;
    public String requestID;
    public int statusCode;

    public String getMessage()
    {
        String s = "";

        if (this.code != null && this.code.length() > 0) {
            s += "Code: " + this.code + " ";
        }

        s += "Message: " + this.chainMessage;

        if (this.detail != null && this.detail.length() > 0) {
            s += " Detail: " + this.detail;
        }

        if (this.requestID != null) {
            s += " Request-ID: " + this.requestID;
        }

        return s;
    }
}
