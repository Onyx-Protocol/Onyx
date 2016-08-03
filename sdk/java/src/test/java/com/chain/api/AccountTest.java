package com.chain.api;

import com.chain.test.BaseTest;
import com.chain.signing.*;

import org.junit.Test;
import static org.junit.Assert.assertEquals;

import java.net.URL;
import java.util.Arrays;

public class AccountTest extends BaseTest {

    @Test public void builderCreate() throws Exception {
        String xpubString = "xpub1234";
        String accountId = "test-account-id";
        String tag = "t1";
        createMockResponse("{\"id\":\"" + accountId + "\",\"quorum\":1, \"xpubs\": [\"" + xpubString + "\"], \"tags\":[\"" + tag + "\"]}");

        KeyHandle testKey = new KeyHandle(xpubString, new URL("http://example.com"));

        Account acc = new Account.Builder()
                .setId(accountId)
                .addKey(testKey)
                .addTag(tag)
                .setQuorum(1)
                .create(ctx);

        validateRequest("POST", "/create-account");
        assertEquals("Ids match", accountId, acc.id);
        assertEquals("Quorum matches", 1, acc.quorum);
        assertEquals("Xpubs equal", Arrays.asList(xpubString),  acc.xpubs);
        assertEquals("Tags equal", Arrays.asList(tag),  acc.tags);
    }

     @Test public void queryList() throws Exception {
     }

     @Test public void nextPage() throws Exception {
     }

     @Test public void queryFind() throws Exception {
     }
}
