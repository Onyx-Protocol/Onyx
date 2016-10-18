package com.chain.http;

import com.chain.exception.*;
import com.google.gson.*;
import com.squareup.okhttp.*;

import java.io.IOException;
import java.io.Reader;
import java.lang.reflect.Type;
import java.net.*;
import java.util.Arrays;
import java.util.Random;
import java.util.concurrent.TimeUnit;

/**
 * HTTP client used to make requests to the Chain Core server.
 */
public class APIClient {
  private URL baseURL;
  private String credentials;
  private OkHttpClient httpClient;
  /**
   * Specifies the MIME type for HTTP requests.
   */
  public static final MediaType JSON = MediaType.parse("application/json; charset=utf-8");
  /**
   * Serializer object used to serialize/deserialize json requests/responses.
   */
  public static final Gson serializer = new Gson();

  /**
   * Default constructor sets the base URL for the client.
   * @param url location of the Chain Core server
   */
  public APIClient(URL url) {
    this.baseURL = url;
    this.httpClient = new OkHttpClient();
    this.httpClient.setFollowRedirects(false);
    String userinfo = url.getUserInfo();
    if (userinfo != null && !userinfo.isEmpty()) {
      credentials = buildCredentials(userinfo);
    }
  }

  /**
   * Sets the base URL and client access token for the client.
   * @param url location of the Chain Core server
   * @param accessToken client access token for the server
   */
  public APIClient(URL url, String accessToken) {
    this(url);
    credentials = buildCredentials(accessToken);
  }

  /**
   * Pins a public key to the HTTP client.
   * @param provider certificate provider
   * @param subjPubKeyInfoHash public key hash
   */
  public void pinCertificate(String provider, String subjPubKeyInfoHash) {
    CertificatePinner cp =
        new CertificatePinner.Builder().add(provider, subjPubKeyInfoHash).build();
    this.httpClient.setCertificatePinner(cp);
  }

  /**
   * Sets the default connect timeout for new connections. A value of 0 means no timeout.
   * @param timeout the number of time units for the default timeout
   * @param unit the unit of time
   */
  public void setConnectTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setConnectTimeout(timeout, unit);
  }

  /**
   * Sets the default read timeout for new connections. A value of 0 means no timeout.
   * @param timeout the number of time units for the default timeout
   * @param unit the unit of time
   */
  public void setReadTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setReadTimeout(timeout, unit);
  }

  /**
   * Sets the default write timeout for new connections. A value of 0 means no timeout.
   * @param timeout the number of time units for the default timeout
   * @param unit the unit of time
   */
  public void setWriteTimeout(long timeout, TimeUnit unit) {
    this.httpClient.setWriteTimeout(timeout, unit);
  }

  /**
   * Sets the proxy information for the HTTP client.
   * @param proxy proxy object
   */
  public void setProxy(Proxy proxy) {
    this.httpClient.setProxy(proxy);
  }

  public interface ResponseCreator<T> {
    public T create(Response response, Gson deserializer) throws ChainException, IOException;
  }

  public <T> T post(String path, Object body, ResponseCreator<T> respCreator)
      throws ChainException {
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
        return respCreator.create(resp, serializer);
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

  private static final Random randomGenerator = new Random();
  private static final int MAX_RETRIES = 10;
  private static final int RETRY_BASE_DELAY_MILLIS = 40;
  private static final int RETRY_MAX_DELAY_MILLIS = 4000;

  private static int retryDelayMillis(int retryAttempt) {
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
