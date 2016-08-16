package com.chain.http;

import com.chain.exception.ChainException;

import java.lang.reflect.Type;
import java.net.*;
import java.util.concurrent.TimeUnit;

/**
 * The Context object contains all information necessary to
 * perform an HTTP request against a remote API.
 */
public class Context {

    private URL url;
    private APIClient httpClient;

    /**
     * Create a new http Context object
     *
     * @param url The URL of the Chain Core or HSM. Includes basic authentication
     *            in the URL string i.e. https://u:p@api.chain.com
     */
    public Context(URL url) {
        this.url = url;
        this.httpClient = new APIClient(url);
    }

    /**
     * Perform a single HTTP POST request against the API for a specific action.
     *
     * @param action The requested API action
     * @param body Body payload sent to the API as JSON
     * @param tClass Type of object to be deserialized from the repsonse JSON
     */
    public <T> T request(String action, Object body, Type tClass)
    throws ChainException {
        return httpClient.post(action, APIClient.serializer.toJson(body), tClass);
    }

    public URL getUrl() { return this.url; }

    public void setConnectTimeout(long timeout, TimeUnit unit) {
        this.httpClient.setConnectTimeout(timeout, unit);
    }

    public void setReadTimeout(long timeout, TimeUnit unit) {
        this.httpClient.setReadTimeout(timeout, unit);
    }

    public void setWriteTimeout(long timeout, TimeUnit unit) {
        this.httpClient.setWriteTimeout(timeout, unit);
    }
}
