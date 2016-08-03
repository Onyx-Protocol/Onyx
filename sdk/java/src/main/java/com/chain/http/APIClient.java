package com.chain.http;

import com.chain.exception.*;
import com.fatboyindustrial.gsonjavatime.Converters;
import com.google.gson.*;
import com.google.gson.reflect.TypeToken;
import com.squareup.okhttp.*;

import java.io.IOException;
import java.lang.reflect.Type;
import java.net.*;
import java.util.HashMap;
import java.util.Random;
import java.util.concurrent.TimeUnit;

public class APIClient {
    private URL baseURL;
    private OkHttpClient httpClient;
    public static final MediaType JSON = MediaType.parse("application/json; charset=utf-8");
    public static final Gson serializer = Converters
        .registerOffsetDateTime(new GsonBuilder())
        .registerTypeHierarchyAdapter(byte[].class, new GsonByteHex())
        .create();

    public APIClient(URL url) {
        this.baseURL = url;
        this.httpClient = new OkHttpClient();
        this.httpClient.setFollowRedirects(false);
    }

    public void pinCertificate(String provider, String subjPubKeyInfoHash) {
        CertificatePinner cp = new CertificatePinner.Builder()
            .add(provider, subjPubKeyInfoHash)
            .build();
        this.httpClient.setCertificatePinner(cp);
    }

    /**
     * Sets the default connect timeout for new connections. A value of 0 means
     * no timeout.
     */
    public void setConnectTimeout(long timeout, TimeUnit unit) {
        this.httpClient.setConnectTimeout(timeout, unit);
    }

    /**
     * Sets the default read timeout for new connections. A value of 0 means no
     * timeout.
     */
    public void setReadTimeout(long timeout, TimeUnit unit) {
        this.httpClient.setReadTimeout(timeout, unit);
    }

    /**
     * Sets the default write timeout for new connections. A value of 0 means no
     * timeout.
     */
    public void setWriteTimeout(long timeout, TimeUnit unit) {
        this.httpClient.setWriteTimeout(timeout, unit);
    }

    public void setProxy(Proxy proxy) { this.httpClient.setProxy(proxy); }

    public <T> T post(String path, String body, Type tClass)
    throws ChainException {
        // Build the request once at the beginning.
        Request req;
        RequestBody requestBody = (body == null) ? null : RequestBody.create(this.JSON, body);
        try {
            Request.Builder reqBuilder = new Request.Builder()
                // TODO: include version string in User-Agent when availabe
                .header("User-Agent", "chain-sdk-java")
                .header("Authorization", this.credentials())
                .url(this.url(path))
                .method("POST", requestBody);

             req = reqBuilder.build();
        } catch (MalformedURLException ex) {
            throw new BadURLException(ex.getMessage());
        }

        ChainException exception = null;
        for(int attempt = 1; attempt - 1 <= MAX_RETRIES; attempt++) {
            // Wait between retrys. The first attempt will not wait at all.
            if (attempt > 1) {
                int delayMillis = retryDelayMillis(attempt-1);
                try {
                    TimeUnit.MILLISECONDS.sleep(delayMillis);
                } catch (InterruptedException e) {}
            }

            try {
                Response resp = this.checkError(this.httpClient.newCall(req).execute());
                try {
                    return this.serializer.fromJson(resp.body().charStream(), tClass);
                } catch (IOException ex) {
                    throw new HTTPException(ex.getMessage());
                }
            } catch(IOException ex) {
                // The OkHttp library already performs retries for most
                // I/O-related errors. We can add retries here too if this
                // becomes a problem.
                throw new HTTPException(ex.getMessage());
            } catch (ConnectivityException ex) {
                // ConnectivityExceptions are always retriable.
                exception = ex;
            } catch (APIException ex) {
                // Check if this http status code is retriable.
                if (!isRetriableStatusCode(ex.statusCode)) {
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

    public static int retryDelayMillis(int retryAttempt) {
        // Calculate the max delay as base * 2 ^ (retryAttempt - 1).
        int max = RETRY_BASE_DELAY_MILLIS * (1 << (retryAttempt-1));
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

    private Response checkError(Response response)
    throws ChainException {
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
                HashMap<String, String> msg = this.serializer.fromJson(
                        response.body().charStream(),
                        new TypeToken<HashMap<String, String>>(){}.getType()
                );
                throw new APIException(
                        msg.get("code"),
                        msg.get("message"),
                        msg.get("detail"),
                        rid,
                        response.code()
                );
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

    private String credentials() {
        String userInfo = this.baseURL.getUserInfo();
        String user = "";
        String pass = "";
        if (userInfo != null) {
            String[] parts = userInfo.split(":");
            if (parts.length >= 1) {
                user = parts[0];
            }
            if (parts.length >= 2) {
                pass = parts[1];
            }
        }
        return Credentials.basic(user, pass);
    }

    private static class GsonByteHex implements JsonSerializer<byte[]>, JsonDeserializer<byte[]> {
        final private static char[] hexArray = "0123456789abcdef".toCharArray();

        private static char[] toHex(byte[] data) {
            char[] res = new char[data.length * 2];
            for (int i = 0; i < data.length; i++) {
                int v = data[i] & 0xFF;
                res[i * 2] = hexArray[v >>> 4];
                res[i * 2 + 1] = hexArray[v & 0x0F];
            }
            return res;
        }

        private static byte[] fromHex(char[] src)
        throws Exception {
            byte[] res = new byte[src.length / 2];
            for (int i = 0; i < src.length; i += 2) {
                res[i / 2] = (byte) ((Character.digit(src[i], 16) << 4) + Character.digit(src[i+1], 16));
            }
            return res;
        }

        public JsonElement serialize(byte[] data, Type t, JsonSerializationContext c) {
            return new JsonPrimitive(new String(toHex(data)));
        }

        public byte[] deserialize(JsonElement json, Type t, JsonDeserializationContext c)
        throws JsonParseException {
            char[] src = json.getAsString().toCharArray();
            try {
                return fromHex(src);
            } catch (Exception e) {
                throw new JsonParseException(e);
            }
        }
    }
}
