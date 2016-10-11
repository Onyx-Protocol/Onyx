package com.chain.http;

import com.chain.exception.*;
import com.fatboyindustrial.gsonjavatime.Converters;
import com.google.gson.*;
import com.google.gson.reflect.TypeToken;
import com.squareup.okhttp.*;

import java.io.IOException;
import java.lang.reflect.Type;
import java.net.*;
import java.util.Arrays;
import java.util.List;
import java.util.HashMap;
import java.util.Random;
import java.util.concurrent.TimeUnit;

public class APIClient {
  private URL baseURL;
  private String credentials;
  private OkHttpClient httpClient;
  public static final MediaType JSON = MediaType.parse("application/json; charset=utf-8");
  public static final Gson serializer =
      Converters.registerOffsetDateTime(new GsonBuilder()).create();

  public APIClient(URL url) {
    this.baseURL = url;
    this.httpClient = new OkHttpClient();
    this.httpClient.setFollowRedirects(false);
    String userinfo = url.getUserInfo();
    if (userinfo != null && !userinfo.isEmpty()) {
      credentials = buildCredentials(userinfo);
    }
  }

  public APIClient(URL url, String accessToken) {
    this(url);
    credentials =  buildCredentials(accessToken);
  }

  public void pinCertificate(String provider, String subjPubKeyInfoHash) {
    CertificatePinner cp =
        new CertificatePinner.Builder().add(provider, subjPubKeyInfoHash).build();
    this.httpClient.setCertificatePinner(cp);
  }

  /**
   * Sets the default connect timeout for new connections. A value of 0 means
   * no timeout.
   * @param timeout the number of time units for the default timeout
   * @param unit the unit of time
   */
  public void setConnectTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setConnectTimeout(timeout, unit);
  }

  /**
   * Sets the default read timeout for new connections. A value of 0 means no
   * timeout.
   * @param timeout the number of time units for the default timeout
   * @param unit the unit of time
   */
  public void setReadTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setReadTimeout(timeout, unit);
  }

  /**
   * Sets the default write timeout for new connections. A value of 0 means no
   * timeout.
   * @param timeout the number of time units for the default timeout
   * @param unit the unit of time
   */
  public void setWriteTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setWriteTimeout(timeout, unit);
  }

  public void setProxy(Proxy proxy) {
    this.httpClient.setProxy(proxy);
  }

  public <T> T post(String path, Object body, Type tClass) throws ChainException {
    return post(path, body, tClass, false);
  }

  public <T> T post(String path, Object body, Type tClass, boolean singletonBatch)
      throws ChainException {
    if (singletonBatch && body != null) {
      body = Arrays.asList(body);
    }

    RequestBody requestBody = RequestBody.create(this.JSON, serializer.toJson(body));
    Request req;

    try {
      Request.Builder builder =
          new Request.Builder()
              // TODO: include version string in User-Agent when available
              .header("User-Agent", "chain-sdk-java")
              .url(this.url(path))
              .method("POST", requestBody);
      if (credentials != null) {
        builder = builder.header("Authorization", credentials);
      }
      req = builder.build();
    } catch (MalformedURLException ex) {
      throw new BadURLException(ex.getMessage());
    }

    ChainException exception = null;
    for (int attempt = 1; attempt - 1 <= MAX_RETRIES; attempt++) {
      // Wait between retrys. The first attempt will not wait at all.
      if (attempt > 1) {
        int delayMillis = retryDelayMillis(attempt - 1);
        try {
          TimeUnit.MILLISECONDS.sleep(delayMillis);
        } catch (InterruptedException e) {
        }
      }

      try {
        Response resp = this.checkError(this.httpClient.newCall(req).execute());
        try {
          if (singletonBatch) {
            return deserializeSingletonBatchResponse(resp, tClass);
          }
          return this.serializer.fromJson(resp.body().charStream(), tClass);
        } catch (IOException ex) {
          throw new HTTPException(ex.getMessage());
        }
      } catch (IOException ex) {
        // The OkHttp library already performs retries for most
        // I/O-related errors. We can add retries here too if this
        // becomes a problem.
        throw new HTTPException(ex.getMessage());
      } catch (ConnectivityException ex) {
        // ConnectivityExceptions are always retriable.
        exception = ex;
      } catch (APIException ex) {
        // Check if this error is retriable (either it's a status code that's
        // always retriable or the error is explicitly marked as temporary.
        if (!isRetriableStatusCode(ex.statusCode) && !ex.temporary) {
          throw ex;
        }
        exception = ex;
      }
    }
    throw exception;
  }

  private <T> T deserializeSingletonBatchResponse(Response response, Type tClass)
      throws ChainException, IOException {
    // The response should be a JSON array with a single item. Since we're in a
    // generic method, and it's difficult to force Gson to deserialize into
    // List<T>, we'll do some manual string munging to get a singleton object.
    String body = new String(response.body().bytes()).trim();
    if (body.charAt(0) != '[' || body.charAt(body.length() - 1) != ']') {
      throw new ChainException("Response not an array");
    }
    body = body.substring(1, body.length() - 1);

    // Check for an error in the response
    APIException err = serializer.fromJson(body, APIException.class);
    if (err.code != null) {
      err.requestId = response.headers().get("Chain-Request-ID");
      throw err;
    }

    return serializer.fromJson(body, tClass);
  }

  private static final Random randomGenerator = new Random();
  private static final int MAX_RETRIES = 10;
  private static final int RETRY_BASE_DELAY_MILLIS = 40;
  private static final int RETRY_MAX_DELAY_MILLIS = 4000;

  public static int retryDelayMillis(int retryAttempt) {
    // Calculate the max delay as base * 2 ^ (retryAttempt - 1).
    int max = RETRY_BASE_DELAY_MILLIS * (1 << (retryAttempt - 1));
    max = Math.min(max, RETRY_MAX_DELAY_MILLIS);

    // To incorporate jitter, use a pseudorandom delay between [1, max] millis.
    return randomGenerator.nextInt(max) + 1;
  }

  private static final int[] RETRIABLE_STATUS_CODES = {
    408, // Request Timeout
    429, // Too Many Requests
    500, // Internal Server Error
    502, // Bad Gateway
    503, // Service Unavailable
    504, // Gateway Timeout
    509, // Bandwidth Limit Exceeded
  };

  private static boolean isRetriableStatusCode(int statusCode) {
    for (int i = 0; i < RETRIABLE_STATUS_CODES.length; i++) {
      if (RETRIABLE_STATUS_CODES[i] == statusCode) {
        return true;
      }
    }
    return false;
  }

  private Response checkError(Response response) throws ChainException {
    String rid = response.headers().get("Chain-Request-ID");
    if (rid == null || rid.length() == 0) {
      // Header field Chain-Request-ID is set by the backend
      // API server. If this field is set, then we can expect
      // the body to be well-formed JSON. If it's not set,
      // then we are probably talking to a gateway or proxy.
      throw new ConnectivityException(response);
    }

    if ((response.code() / 100) != 2) {
      try {
        APIException err = serializer.fromJson(response.body().charStream(), APIException.class);
        if (err.code != null) {
          err.requestId = rid;
          err.statusCode = response.code();
          throw err;
        }
      } catch (IOException ex) {
        throw new JSONException("Unable to read body. " + ex.getMessage(), rid);
      }
    }
    return response;
  }

  private URL url(String path) throws MalformedURLException {
    try {
      URI u = new URI(this.baseURL.toString() + "/" + path);
      u = u.normalize();
      return new URL(u.toString());
    } catch (URISyntaxException e) {
      throw new MalformedURLException();
    }
  }

  private static String buildCredentials(String accessToken) {
    String user = "";
    String pass = "";
    if (accessToken != null) {
      String[] parts = accessToken.split(":");
      if (parts.length >= 1) {
        user = parts[0];
      }
      if (parts.length >= 2) {
        pass = parts[1];
      }
    }
    return Credentials.basic(user, pass);
  }
}
