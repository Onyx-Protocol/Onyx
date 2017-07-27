package com.chain.analytics;

import org.junit.Test;

import java.util.Arrays;
import java.util.Collections;

import static junit.framework.TestCase.assertEquals;

public class SchemaTest {

  private static final Schema oneColumn =
      new Schema.Builder("transactions")
          .addColumn("id", new OracleTypes.Varchar2(32))
          .setPrimaryKey(Collections.singletonList("id"))
          .build();
  private static final Schema multipleColumns =
      new Schema.Builder("transaction_outputs")
          .addColumn("transaction_id", new OracleTypes.Varchar2(32))
          .addColumn("index", new OracleTypes.BigInteger())
          .addColumn("output_id", new OracleTypes.Varchar2(32))
          .addUniqueConstraint(Arrays.asList("transaction_id", "index"))
          .setPrimaryKey(Collections.singletonList("output_id"))
          .build();

  @Test
  public void testOneColumnSchemaDDL() {
    final String ddl = oneColumn.getDDLStatement();
    assertEquals(
        "CREATE TABLE TRANSACTIONS (\n"
            + "  \"ID\" VARCHAR2(32),\n"
            + "  CONSTRAINT transactions_pk PRIMARY KEY (\"ID\"))",
        ddl);
  }

  @Test
  public void testMultipleColumnSchemaDDL() {
    final String ddl = multipleColumns.getDDLStatement();
    assertEquals(
        "CREATE TABLE TRANSACTION_OUTPUTS (\n"
            + "  \"TRANSACTION_ID\" VARCHAR2(32),\n"
            + "  \"INDEX\" NUMBER(20),\n"
            + "  \"OUTPUT_ID\" VARCHAR2(32),\n"
            + "  CONSTRAINT transaction_id_index_u UNIQUE (\"TRANSACTION_ID\", \"INDEX\"),\n"
            + "  CONSTRAINT transaction_outputs_pk PRIMARY KEY (\"OUTPUT_ID\"))",
        ddl);
  }

  @Test
  public void testMultipleColumnSchemaInsert() {
    final String insertQuery = multipleColumns.getInsertStatement();
    assertEquals(
        "INSERT INTO TRANSACTION_OUTPUTS\n"
            + "(\"TRANSACTION_ID\", \"INDEX\", \"OUTPUT_ID\")\n"
            + "VALUES(?, ?, ?)",
        insertQuery);
  }
}
