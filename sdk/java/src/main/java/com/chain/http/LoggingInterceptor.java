package com.chain.http;

import com.squareup.okhttp.Interceptor;
import com.squareup.okhttp.Request;
import com.squareup.okhttp.Response;
import okio.BufferedSink;
import okio.BufferedSource;

import java.io.IOException;
import java.io.OutputStream;
import java.nio.charset.Charset;

/**
 * The LoggingInterceptor object logs http requests given
 * an output stream.
 */
public class LoggingInterceptor implements Interceptor {
  private Level level;
  private OutputStream logger;

  public enum Level {
    ALL,
    ERRORS,
    NONE,
  }

  public LoggingInterceptor(OutputStream logger, Level logAllRequests) {
    this.logger = logger;
    this.level = logAllRequests;
  }

  @Override
  public Response intercept(Interceptor.Chain chain) throws IOException {
    Request request = chain.request();
    Response response = chain.proceed(request);

    boolean isError = (response.code() / 100) == 5 || (response.code() / 100) == 4;
    if ((isError && level == level.ERRORS) || level == level.ALL) {
      logRequestData(request, response);
    }

    return response;
  }

  public void logRequestData(Request request, Response response) throws IOException {
    String reqid = response.header("Chain-Request-Id");

    String requestBody;
    try {
      BufferedSink reqBodySink = new okio.Buffer();
      request.body().writeTo(reqBodySink);
      requestBody = new String(reqBodySink.buffer().readByteArray());
    } catch (IOException e) {
      requestBody = "Unable to read request body.";
    }

    String label = "chain-request";
    if (response.code() / 100 == 5) {
      label = "chain-error";
    }

    BufferedSource source = response.body().source();
    source.request(Long.MAX_VALUE);

    logger.write(
        String.format(
                "%s:\n\treqid=%s\n\turl=%s\n\tcode=%d\n\trequest=%s\n\tresponse=%s\n",
                label,
                reqid,
                request.urlString(),
                response.code(),
                requestBody,
                source.buffer().clone().readString(Charset.forName("UTF-8")))
            .getBytes());
  }
}
