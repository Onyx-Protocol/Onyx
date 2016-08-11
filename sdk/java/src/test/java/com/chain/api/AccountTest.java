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

        String testKey = "7abdd659a569d566ffe2bc2e4536d6fa07b8bf4bf87ef0bf760c9363d85fb4e3de69d25e0bc158de9b5684d76a7e40f2b7c537107d6c5b2a07c42cc923993a77";

        Map<String, Object> tags = new HashMap<>();
        tags.put("hello", Arrays.asList("this", "is", "an", "array"));

        Account acc = new Account.Builder()
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
