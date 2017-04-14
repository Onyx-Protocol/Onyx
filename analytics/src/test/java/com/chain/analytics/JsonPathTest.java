package com.chain.analytics;

import com.chain.api.Transaction;

import java.util.Arrays;
import java.util.Map;
import java.util.TreeMap;
import org.junit.Test;
import static junit.framework.TestCase.assertEquals;
import static junit.framework.TestCase.assertNull;

public class JsonPathTest {

  @Test
  public void testExtract() {
    Map<String, Object> map = new TreeMap<String, Object>();
    Map<String, Object> innerMap = new TreeMap<String, Object>();
    innerMap.put("id", "abc");
    innerMap.put("account_number", 123);
    map.put("account", innerMap);
    map.put("id", 12345);

    final Transaction tx = new Transaction();
    tx.referenceData = map;

    JsonPath pathId = new JsonPath(Arrays.asList("reference_data", "id"));
    JsonPath pathAccountId = new JsonPath(Arrays.asList("reference_data", "account", "id"));
    JsonPath pathAccountAccountNumber =
        new JsonPath(Arrays.asList("reference_data", "account", "account_number"));

    assertEquals(pathId.extract(tx), 12345);
    assertEquals(pathAccountId.extract(tx), "abc");
    assertEquals(pathAccountAccountNumber.extract(tx), 123);
  }
}
