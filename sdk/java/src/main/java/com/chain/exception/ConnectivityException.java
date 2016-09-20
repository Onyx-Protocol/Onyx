package com.chain.exception;

import com.squareup.okhttp.Response;

import java.io.IOException;

public class ConnectivityException extends ChainException {
  public ConnectivityException(Response resp) {
    super(formatMessage(resp));
  }

  private static String formatMessage(Response resp) {
    String s =
        "Response HTTP header field Chain-Request-ID is unset. There may be network issues. Please check your local network settings.";
    // TODO(kr): include client-generated reqid here once we have that.
    String body;
    try {
      body = resp.body().string();
    } catch (IOException ex) {
      body = "[unable to read response body: " + ex.toString() + "]";
    }
    return String.format("%s status=%d body=%s", s, resp.code(), body);
  }
}
