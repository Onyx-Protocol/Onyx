package com.chain.signing;

import com.chain.test.BaseTest;
import com.chain.http.*;

import org.junit.Test;
import static org.junit.Assert.assertEquals;

import java.net.URL;

public class KeyHandleTest extends BaseTest {

    @Test public void builderCreate() throws Exception {
        createMockResponse("{\"xpub\":\"xpub1234\"}");

        Context hsmCtx = new Context(new URL(server.url("").toString()));
        KeyHandle keyHandle = new KeyHandle.Builder()
            .create(hsmCtx);

        validateRequest("POST", "/create-key");
        assertEquals("xpub", "xpub1234",  keyHandle.getXPub());
    }

    @Test public void queryList() throws Exception {

    }
}
