package analytics;

import java.util.Arrays;
import org.junit.Test;
import static junit.framework.TestCase.assertEquals;

public class SchemaTest {

    static final Schema oneColumn = new Schema.Builder("transactions")
            .addColumn("id", new Schema.Varchar2(32))
            .setPrimaryKey(Arrays.asList("id"))
            .build();
    static final Schema multipleColumns = new Schema.Builder("transaction_outputs")
            .addColumn("transaction_id", new Schema.Varchar2(32))
            .addColumn("index", new Schema.Integer())
            .addColumn("output_id", new Schema.Varchar2(32))
            .addUniqueConstraint(Arrays.asList("transaction_id", "index"))
            .setPrimaryKey(Arrays.asList("output_id"))
            .build();

    @Test
    public void testOneColumnSchemaDDL() {
        final String ddl = oneColumn.getDDLStatement();
        assertEquals("CREATE TABLE TRANSACTIONS (\n" +
                "  \"ID\" VARCHAR2(32),\n" +
                "  CONSTRAINT transactions_pk PRIMARY KEY (\"ID\"))", ddl);
    }

    @Test
    public void testMultipleColumnSchemaDDL() {
        final String ddl = multipleColumns.getDDLStatement();
        assertEquals(
                "CREATE TABLE TRANSACTION_OUTPUTS (\n" +
                "  \"TRANSACTION_ID\" VARCHAR2(32),\n" +
                "  \"INDEX\" NUMBER(20),\n" +
                "  \"OUTPUT_ID\" VARCHAR2(32),\n" +
                "  CONSTRAINT transaction_id_index_u UNIQUE (\"TRANSACTION_ID\", \"INDEX\"),\n" +
                "  CONSTRAINT transaction_outputs_pk PRIMARY KEY (\"OUTPUT_ID\"))", ddl);
    }

    @Test
    public void testMultipleColumnSchemaInsert() {
        final String insertQuery = multipleColumns.getInsertStatement();
        assertEquals(
                "INSERT INTO TRANSACTION_OUTPUTS\n" +
                "(\"TRANSACTION_ID\", \"INDEX\", \"OUTPUT_ID\")\n" +
                "VALUES(?, ?, ?)", insertQuery);
    }

}
