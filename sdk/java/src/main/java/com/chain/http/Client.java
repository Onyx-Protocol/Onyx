package com.chain.http;

import com.chain.exception.*;

import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.lang.reflect.Type;
import java.net.*;
import java.util.Random;
import java.util.concurrent.TimeUnit;
import java.util.List;

import com.google.gson.Gson;

import com.squareup.okhttp.CertificatePinner;
import com.squareup.okhttp.ConnectionPool;
import com.squareup.okhttp.Credentials;
import com.squareup.okhttp.MediaType;
import com.squareup.okhttp.OkHttpClient;
import com.squareup.okhttp.Request;
import com.squareup.okhttp.RequestBody;
import com.squareup.okhttp.Response;

/**
 * The Client object contains all information necessary to
 * perform an HTTP request against a remote API. Typically,
 * an application will have a client that makes requests to
 * a Chain Core, and a separate Client that makes requests
 * to an HSM server.
 */
public class Client {

  private URL url;
  private String accessToken;
  private OkHttpClient httpClient;
  private static final MediaType JSON = MediaType.parse("application/json; charset=utf-8");
  private static final Gson serializer = new Gson();
  private static String version = "dev"; // updated in the static initializer

  private static class BuildProperties {
    public String version;
  }

  static {
    InputStream in = Client.class.getClassLoader().getResourceAsStream("properties.json");
    if (in != null) {
      InputStreamReader inr = new InputStreamReader(in);
      version = serializer.fromJson(inr, BuildProperties.class).version;
    }
  }

  public Client(Builder builder) {
    this.url = builder.url;
    this.accessToken = builder.accessToken;
    this.httpClient = buildHttpClient(builder);
  }

  /**
   * Create a new http Client object using the default development host URL.
   */
  public Client() {
    this(new Builder());
  }

  /**
   * Create a new http Client object
   *
   * @param url the URL of the Chain Core or HSM
   */
  public Client(String url) throws BadURLException {
    this(new Builder().setURL(url));
  }

  /**
   * Create a new http Client object
   *
   * @param url the URL of the Chain Core or HSM
   */
  public Client(URL url) {
    this(new Builder().setURL(url));
  }

  /**
   * Create a new http Client object
   *
   * @param url the URL of the Chain Core or HSM
   * @param accessToken a Client API access token
   */
  public Client(String url, String accessToken) throws BadURLException {
    this(new Builder().setURL(url).setAccessToken(accessToken));
  }

  /**
   * Create a new http Client object
   *
   * @param url the URL of the Chain Core or HSM
   * @param accessToken a Client API access token
   */
  public Client(URL url, String accessToken) {
    this(new Builder().setURL(url).setAccessToken(accessToken));
  }

  /**
   * Perform a single HTTP POST request against the API for a specific action.
   *
   * @param action The requested API action
   * @param body Body payload sent to the API as JSON
   * @param tClass Type of object to be deserialized from the repsonse JSON
   * @return the result of the post request
   * @throws ChainException
   */
  public <T> T request(String action, Object body, final Type tClass) throws ChainException {
    ResponseCreator<T> rc =
        new ResponseCreator<T>() {
          public T create(Response response, Gson deserializer) throws IOException {
            return deserializer.fromJson(response.body().charStream(), tClass);
          }
        };
    return post(action, body, rc);
  }

  /**
   * Perform a single HTTP POST request against the API for a specific action.
   * Use this method if you want batch semantics, i.e., the endpoint response
   * is an array of valid objects interleaved with arrays, once corresponding to
   * each input object.
   *
   * @param action The requested API action
   * @param body Body payload sent to the API as JSON
   * @param tClass Type of object to be deserialized from the response JSON
   * @param eClass Type of error object to be deserialized from the response JSON
   * @return the result of the post request
   * @throws ChainException
   */
  public <T> BatchResponse<T> batchRequest(
      String action, Object body, final Type tClass, final Type eClass) throws ChainException {
    ResponseCreator<BatchResponse<T>> rc =
        new ResponseCreator<BatchResponse<T>>() {
          public BatchResponse<T> create(Response response, Gson deserializer)
              throws ChainException, IOException {
            return new BatchResponse<>(response, deserializer, tClass, eClass);
          }
        };
    return post(action, body, rc);
  }

  /**
   * Perform a single HTTP POST request against the API for a specific action.
   * Use this method if you want single-item semantics (creating single assets,
   * building single transactions) but the API endpoint is implemented as a
   * batch call.
   *
   * Because request bodies for batch calls do not share a consistent format,
   * this method does not perform any automatic arrayification of outgoing
   * parameters. Remember to arrayify your request objects where appropriate.
   *
   * @param action The requested API action
   * @param body Body payload sent to the API as JSON
   * @param tClass Type of object to be deserialized from the repsonse JSON
   * @return the result of the post request
   * @throws ChainException
   */
  public <T> T singletonBatchRequest(
      String action, Object body, final Type tClass, final Type eClass) throws ChainException {
    ResponseCreator<T> rc =
        new ResponseCreator<T>() {
          public T create(Response response, Gson deserializer) throws ChainException, IOException {
            BatchResponse<T> batch = new BatchResponse<>(response, deserializer, tClass, eClass);

            List<APIException> errors = batch.errors();
            if (errors.size() == 1) {
              // This throw must occur within this lambda in order for APIClient's
              // retry logic to take effect.
              throw errors.get(0);
            }

            List<T> successes = batch.successes();
            if (successes.size() == 1) {
              return successes.get(0);
            }

            // We should never get here, unless there is a bug in either the SDK or
            // API code, causing a non-singleton response.
            throw new ChainException(
                "Invalid singleton repsonse, request ID "
                    + batch.response().headers().get("Chain-Request-ID"));
          }
        };
    return post(action, body, rc);
  }

  /**
   * Returns the base url stored in the client.
   * @return the client's base url
   */
  public URL url() {
    return this.url;
  }

  /**
   * Returns true if a client access token stored in the client.
   * @return a boolean
   */
  public boolean hasAccessToken() {
    return this.accessToken != null && !this.accessToken.isEmpty();
  }

  /**
   * Returns the client access token (possibly null).
   * @return the client access token
   */
  public String accessToken() {
    return accessToken;
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

  /**
   * Defines an interface for deserializing HTTP responses into objects.
   * @param <T> the type of object to return
   */
  public interface ResponseCreator<T> {
    /**
     * Deserializes an HTTP response into a Java object of type T.
     * @param response HTTP response object
     * @param deserializer json deserializer
     * @return an object of type T
     * @throws ChainException
     * @throws IOException
     */
    T create(Response response, Gson deserializer) throws ChainException, IOException;
  }

  /**
   * Builds and executes an HTTP Post request.
   * @param path the path to the endpoint
   * @param body the request body
   * @param respCreator object specifying the response structure
   * @return a response deserialized into type T
   * @throws ChainException
   */
  private <T> T post(String path, Object body, ResponseCreator<T> respCreator)
      throws ChainException {
    RequestBody requestBody = RequestBody.create(this.JSON, serializer.toJson(body));
    Request req;

    try {
      Request.Builder builder =
          new Request.Builder()
              .header("User-Agent", "chain-sdk-java/" + version)
              .url(this.createEndpoint(path))
              .method("POST", requestBody);
      if (hasAccessToken()) {
        builder = builder.header("Authorization", buildCredentials());
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

  private OkHttpClient buildHttpClient(Builder builder) {
    OkHttpClient httpClient = new OkHttpClient();
    httpClient.setFollowRedirects(false);
    httpClient.setReadTimeout(builder.readTimeout, builder.readTimeoutUnit);
    httpClient.setWriteTimeout(builder.writeTimeout, builder.writeTimeoutUnit);
    httpClient.setConnectTimeout(builder.connectTimeout, builder.connectTimeoutUnit);

    httpClient.setConnectionPool(builder.pool);

    if (builder.proxy != null) {
      httpClient.setProxy(builder.proxy);
    }
    if (builder.cp != null) {
      httpClient.setCertificatePinner(builder.cp);
    }

    return httpClient;
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

  private URL createEndpoint(String path) throws MalformedURLException {
    try {
      URI u = new URI(this.url.toString() + "/" + path);
      u = u.normalize();
      return new URL(u.toString());
    } catch (URISyntaxException e) {
      throw new MalformedURLException();
    }
  }

  private String buildCredentials() {
    String user = "";
    String pass = "";
    if (hasAccessToken()) {
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

  private String identifier() {
    if (this.hasAccessToken()) {
      return String.format(
          "%s://%s@%s", this.url.getProtocol(), this.accessToken, this.url.getAuthority());
    }
    return String.format("%s://%s", this.url.getProtocol(), this.url.getAuthority());
  }

  /**
   * Overrides {@link Object#hashCode()}
   * @return the hash code
   */
  @Override
  public int hashCode() {
    return identifier().hashCode();
  }

  /**
   * Overrides {@link Object#equals(Object)}
   * @param o the object to compare
   * @return a boolean specifying equality
   */
  @Override
  public boolean equals(Object o) {
    if (o == null) return false;
    if (!(o instanceof Client)) return false;

    Client other = (Client) o;
    return this.identifier().equals(other.identifier());
  }

  /**
   * A builder class for creating client objects
   */
  public static class Builder {
    private URL url;
    private String accessToken;
    private CertificatePinner cp;
    private long connectTimeout;
    private TimeUnit connectTimeoutUnit;
    private long readTimeout;
    private TimeUnit readTimeoutUnit;
    private long writeTimeout;
    private TimeUnit writeTimeoutUnit;
    private Proxy proxy;
    private ConnectionPool pool;

    public Builder() {
      this.setDefaults();
    }

    private void setDefaults() {
      try {
        this.url = new URL("http://localhost:1999");
      } catch (MalformedURLException e) {
        throw new RuntimeException("invalid default development URL", e);
      }
      this.setReadTimeout(30, TimeUnit.SECONDS);
      this.setWriteTimeout(30, TimeUnit.SECONDS);
      this.setConnectTimeout(30, TimeUnit.SECONDS);
      this.setConnectionPool(50, 2, TimeUnit.MINUTES);
    }

    /**
     * Sets the URL for the client
     * @param url the URL of the Chain Core or HSM
     */
    public Builder setURL(String url) throws BadURLException {
      try {
        this.url = new URL(url);
      } catch (MalformedURLException e) {
        throw new BadURLException(e.getMessage());
      }
      return this;
    }

    /**
     * Sets the URL for the client
     * @param url the URL of the Chain Core or HSM
     */
    public Builder setURL(URL url) {
      this.url = url;
      return this;
    }

    /**
     * Sets the access token for the client
     * @param accessToken The access token for the Chain Core or HSM
     */
    public Builder setAccessToken(String accessToken) {
      this.accessToken = accessToken;
      return this;
    }

    /**
     * Sets the certificate pinner for the client
     * @param provider certificate provider
     * @param subjPubKeyInfoHash public key hash
     */
    public Builder pinCertificate(String provider, String subjPubKeyInfoHash) {
      this.cp = new CertificatePinner.Builder().add(provider, subjPubKeyInfoHash).build();
      return this;
    }

    /**
     * Sets the connect timeout for the client
     * @param timeout the number of time units for the default timeout
     * @param unit the unit of time
     */
    public Builder setConnectTimeout(long timeout, TimeUnit unit) {
      this.connectTimeout = timeout;
      this.connectTimeoutUnit = unit;
      return this;
    }

    /**
     * Sets the read timeout for the client
     * @param timeout the number of time units for the default timeout
     * @param unit the unit of time
     */
    public Builder setReadTimeout(long timeout, TimeUnit unit) {
      this.readTimeout = timeout;
      this.readTimeoutUnit = unit;
      return this;
    }

    /**
     * Sets the write timeout for the client
     * @param timeout the number of time units for the default timeout
     * @param unit the unit of time
     */
    public Builder setWriteTimeout(long timeout, TimeUnit unit) {
      this.writeTimeout = timeout;
      this.writeTimeoutUnit = unit;
      return this;
    }

    /**
     * Sets the proxy for the client
     * @param proxy
     */
    public Builder setProxy(Proxy proxy) {
      this.proxy = proxy;
      return this;
    }

    /**
     * Sets the connection pool for the client
     * @param maxIdle the maximum number of idle http connections in the pool
     * @param timeout the number of time units until an idle http connection in the pool is closed
     * @param unit the unit of time
     */
    public Builder setConnectionPool(int maxIdle, long timeout, TimeUnit unit) {
      this.pool = new ConnectionPool(maxIdle, unit.toMillis(timeout));
      return this;
    }

    /**
     * Builds a client with all of the provided parameters.
     */
    public Client build() {
      return new Client(this);
    }
  }
}
