package com.chain.test;

import com.chain.http.Context;

import com.squareup.okhttp.HttpUrl;
import com.squareup.okhttp.mockwebserver.MockResponse;
import com.squareup.okhttp.mockwebserver.MockWebServer;
import com.squareup.okhttp.mockwebserver.RecordedRequest;

import java.net.MalformedURLException;
import java.net.URL;

import org.junit.BeforeClass;
import static org.junit.Assert.assertEquals;

public abstract class BaseTest {
    public static MockWebServer server;
    public static HttpUrl baseURL;
    public static Context ctx;

    /**
     * Initializes a MockWebServer object and a Chain Context.
     * @throws java.net.MalformedURLException
     */
    @BeforeClass public static void init() throws MalformedURLException {
        server = new MockWebServer();
        baseURL = server.url("");
        ctx = new Context(new URL(baseURL.toString()));
    }

    /**
     * Creates a mock response for the mock web server.
     * @param body a json string representing an actual Chain core response body
     * @return a MockResponse object
     */
    public static MockResponse createMockResponse(String body) {
        MockResponse mr = new MockResponse()
            .addHeader("Chain-Request-Id", "1")
            .addHeader("Content-Type", "application/json")
            .setBody(body);
        server.enqueue(mr);
        return mr;
    }

    /**
     * Validates HTTP requests made by the SDK.
     * @param method the expected HTTP method
     * @param path the expected URI of the HTTP request
     * @throws InterruptedException
     */
    public static void validateRequest(String method, String path) throws InterruptedException {
        RecordedRequest request = server.takeRequest();
        assertEquals("request method", method, request.getMethod());
        assertEquals("request path", path, request.getPath());
    }
}
