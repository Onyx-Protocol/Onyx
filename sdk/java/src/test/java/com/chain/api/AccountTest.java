package com.chain.api;

import com.chain.test.BaseTest;
import com.chain.signing.*;

import org.junit.Test;
import static org.junit.Assert.assertEquals;

import java.net.URL;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

public class AccountTest extends BaseTest {

    @Test public void builderCreate() throws Exception {
        String xpubString = "xpub1234";
        String accountId = "test-account-id";
        String tag = "t1";
        createMockResponse("{\"id\":\"" + accountId + "\",\"quorum\":1, \"xpubs\": [\"" + xpubString + "\"], \"tags\":{\"hello\": [\"this\", \"is\", \"an\", \"array\"]}}");

        KeyHandle testKey = new KeyHandle(xpubString, new URL("http://example.com"));

        Map<String, Object> tags = new HashMap<>();
        tags.put("hello", Arrays.asList("this", "is", "an", "array"));

        Account acc = new Account.Builder()
                .setId(accountId)
                .addXpub(testKey)
                .setTags(tags)
                .setQuorum(1)
                .create(ctx);

        validateRequest("POST", "/create-account");
        assertEquals("Ids match", accountId, acc.id);
        assertEquals("Quorum matches", 1, acc.quorum);
        assertEquals("Xpubs equal", Arrays.asList(xpubString),  acc.xpubs);
        assertEquals("Tags equal", tags,  acc.tags);
    }

     @Test public void queryList() throws Exception {
     }

     @Test public void nextPage() throws Exception {
     }

     @Test public void queryFind() throws Exception {
     }
}
