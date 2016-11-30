package com.chain.http;

import java.io.InputStream;
import java.io.InputStreamReader;
import java.lang.reflect.Type;
import java.net.*;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Executor;

import com.chain.exception.BadURLException;
import com.chain.proto.*;

import com.google.common.reflect.TypeToken;
import io.grpc.*;

import com.google.gson.Gson;

/**
 * The Client object contains all information necessary to
 * perform an HTTP request against a remote API. Typically,
 * an application will have a client that makes requests to
 * a Chain Core, and a separate Client that makes requests
 * to an HSM server.
 */
public class Client {

  private String accessToken;
  private String accessUser;
  private String accessPass;
  private static final Gson serializer = new Gson();
  private static String version = "dev"; // updated in the static initializer
  private ManagedChannel channel;
  private AppGrpc.AppBlockingStub appStub;
  private HSMGrpc.HSMBlockingStub hsmStub;

  private static final Metadata.Key USERNAME =
      Metadata.Key.of("username", Metadata.ASCII_STRING_MARSHALLER);
  private static final Metadata.Key PASSWORD =
      Metadata.Key.of("password", Metadata.ASCII_STRING_MARSHALLER);

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

  public Client() throws BadURLException {
    this("http://localhost:1999");
  }

  public Client(URL url) {
    init(url, null);
  }

  public Client(String urlString) throws BadURLException {
    try {
      init(new URL(urlString), null);
    } catch (MalformedURLException e) {
      throw new BadURLException(e.getMessage());
    }
  }

  public Client(URL url, String accessToken) {
    init(url, accessToken);
  }

  public Client(String urlString, String accessToken) throws BadURLException {
    try {
      init(new URL(urlString), accessToken);
    } catch (MalformedURLException e) {
      throw new BadURLException(e.getMessage());
    }
  }

  private void init(URL url, String accessToken) {
    channel =
        ManagedChannelBuilder.forAddress(url.getHost(), url.getPort())
            .usePlaintext(url.getProtocol().equals("http"))
            .build();

    this.accessToken = accessToken;
    this.appStub = (AppGrpc.AppBlockingStub) initStub(AppGrpc.newBlockingStub(channel));
    this.hsmStub = (HSMGrpc.HSMBlockingStub) initStub(HSMGrpc.newBlockingStub(channel));

    if (accessToken != null && !accessToken.isEmpty()) {
      String[] parts = accessToken.split(":");
      if (parts.length == 2) {
        this.accessUser = parts[0];
        this.accessPass = parts[1];
      }
    }
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

  public AppGrpc.AppBlockingStub app() {
    return this.appStub;
  }

  public HSMGrpc.HSMBlockingStub hsm() {
    return this.hsmStub;
  }

  private io.grpc.stub.AbstractStub<?> initStub(io.grpc.stub.AbstractStub<?> stub) {
    return stub.withCompression("gzip")
        .withCallCredentials(
            new CallCredentials() {
              @Override
              public void applyRequestMetadata(
                  MethodDescriptor<?, ?> methodDescriptor,
                  Attributes attributes,
                  Executor executor,
                  MetadataApplier metadataApplier) {

                Metadata authData = new Metadata();
                if (accessUser != null && !accessUser.isEmpty()) {
                  authData.put(USERNAME, accessUser);
                  authData.put(PASSWORD, accessUser);
                }
                metadataApplier.apply(authData);
              }
            });
  }

  public byte[] serialize(Object obj) {
    return serializer.toJson(obj).getBytes();
  }

  public Map<String, Object> deserialize(String data) {
    Type type = new TypeToken<HashMap<String, Object>>() {}.getType();
    return serializer.fromJson(data, type);
  }

  public <T> T deserialize(String data, Class<T> type) {
    return serializer.fromJson(data, type);
  }

  public <T> T deserialize(String data, Type type) {
    return serializer.fromJson(data, type);
  }
}
