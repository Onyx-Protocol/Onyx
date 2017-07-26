package com.chain.analytics;

import org.junit.Test;

import java.io.Reader;
import java.io.StringReader;
import java.util.Arrays;
import java.util.Collections;

import static junit.framework.TestCase.assertEquals;

public class ConfigTest {
  @Test
  public void loadFromJSON() throws Config.InvalidConfigException {
    Reader reader =
        new StringReader(
            "{\n"
                + "\"transactionColumns\": [\n"
                + "{\n"
                + "\"name\": \"internal_tx_id\",\n"
                + "\"type\": \"varchar(50)\",\n"
                + "\"path\": \"reference_data.tx_id\"\n"
                + "}\n"
                + "]}");
    final Config config = Config.readFromJSON(reader);
    assertEquals(1, config.transactionColumns.size());
    final Config.CustomColumn col = config.transactionColumns.get(0);
    assertEquals("internal_tx_id", col.name);
    assertEquals("varchar(50)", col.type.toString());
    assertEquals("reference_data.tx_id", col.jsonPath.toString());
  }
}
