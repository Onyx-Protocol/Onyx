package com.chain.analytics;

import org.junit.Test;
import static junit.framework.TestCase.assertEquals;
import static junit.framework.TestCase.assertNull;

public class OracleTypesTest {
  @Test
  public void testParseInvalidTypes() {
    assertNull(OracleTypes.parse("bytea"));
    assertNull(OracleTypes.parse("integer"));
    assertNull(OracleTypes.parse("nvarchar2(100)"));
    assertNull(OracleTypes.parse("blob(100)"));
    assertNull(OracleTypes.parse("blob((100))"));
    assertNull(OracleTypes.parse("varchar((100))"));
    assertNull(OracleTypes.parse("varchar(4001)"));
  }

  @Test
  public void testParseToString() {
    assertEquals("bigint", OracleTypes.parse("bigint").toString());
    assertEquals("blob", OracleTypes.parse("blob").toString());
    assertEquals("boolean", OracleTypes.parse("boolean").toString());
    assertEquals("clob", OracleTypes.parse("clob").toString());
    assertEquals("timestamp", OracleTypes.parse("timestamp").toString());
    assertEquals("varchar(64)", OracleTypes.parse("varchar(64)").toString());
    assertEquals("varchar(100)", OracleTypes.parse("varchar(100)").toString());
  }
}
