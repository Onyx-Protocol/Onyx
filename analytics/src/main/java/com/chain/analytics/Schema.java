package analytics;

import java.util.*;
import java.sql.Types;

/**
 * Schema represents the schema of an Oracle database table. Schemas
 * are generated based on the Chain Analytics configuration. Once
 * constructed, a Schema is immutable.
 */
public class Schema {
  String mName;
  List<Column> mColumns;
  List<String> mPrimaryKey;
  List<List<String>> mUniqueConstraints;

  /**
   * Column represents a single column in a table.
   */
  public static class Column {
    String name;
    SQLType type;

    public Column(final String name, final SQLType type) {
      this.name = name;
      this.type = type;
    }
  }

  /**
   * SQLType describes a SQL type that can be used when
   * constructing a DDL query.
   */
  public interface SQLType {
    String toString();

    int getType();
  }

  /**
   * Builder implements the builder pattern for a Schema.
   */
  public static class Builder {
    private String mName;
    private List<Column> mColumns;
    private List<String> mPrimaryKey;
    private List<List<String>> mUniqueConstraints;

    /**
     * Constructor for a builder.
     *
     * @param name the name of the table
     */
    public Builder(final String name) {
      mName = name;
      mColumns = new ArrayList<>();
      mPrimaryKey = Collections.emptyList();
      mUniqueConstraints = new ArrayList<>();
    }

    /**
     * Adds a column to the table.
     *
     * @param name the name of the SQL column
     * @param typ  the Oracle SQL type of the column
     */
    public Builder addColumn(final String name, final SQLType typ) {
      mColumns.add(new Column(name, typ));
      return this;
    }

    /**
     * Adds a uniqueness constraint on the provided columns.
     *
     * @param columns a list of the columns that the uniqueness
     *                constraint covers.
     */
    public Builder addUniqueConstraint(final List<String> columns) {
      mUniqueConstraints.add(columns);
      return this;
    }

    /**
     * Adds a primary key constraint on the provided columns.
     *
     * @param columns a list of one or more columns that form
     *                the table's primary key.
     */
    public Builder setPrimaryKey(final List<String> columns) {
      mPrimaryKey = columns;
      return this;
    }

    /**
     * Constructs the schema.
     *
     * @return the built, immutable Schema
     */
    public Schema build() {
      final Schema schema = new Schema();
      schema.mName = mName;
      schema.mColumns = Collections.unmodifiableList(mColumns);
      schema.mPrimaryKey = Collections.unmodifiableList(mPrimaryKey);
      schema.mUniqueConstraints = Collections.unmodifiableList(mUniqueConstraints);
      return schema;
    }
  }

  /**
   * Constructs a CREATE TABLE statement for the schema.
   * @return a CREATE TABLE DDL statement
   */
  public String getDDLStatement() {
    final StringBuilder sb = new StringBuilder();
    sb.append("CREATE TABLE ").append(mName.toUpperCase()).append(" (");

    String sep = "";
    for (final Column col : mColumns) {
      sb.append(sep)
          .append("\n")
          .append("  ")
          .append("\"")
          .append(col.name.toUpperCase())
          .append("\"")
          .append(" ")
          .append(col.type.toString());

      // use a comma separator before every column after the first.
      sep = ",";
    }

    for (final List<String> cols : mUniqueConstraints) {
      sb.append(",\n  CONSTRAINT ").append(String.join("_", cols)).append("_u UNIQUE (");
      sep = "";
      for (final String col : cols) {
        sb.append(sep).append("\"").append(col.toUpperCase()).append("\"");
        sep = ", ";
      }
      sb.append(")");
    }

    if (!mPrimaryKey.isEmpty()) {
      sb.append(",\n  CONSTRAINT ").append(mName).append("_pk PRIMARY KEY (");
      sep = "";
      for (final String keyCol : mPrimaryKey) {
        sb.append(sep).append("\"").append(keyCol.toUpperCase()).append("\"");
        sep = ", ";
      }
      sb.append(")");
    }

    sb.append(")");
    return sb.toString();
  }

  /**
   * Constructs the beginning of an INSERT statement for the
   * table.
   */
  public String getInsertStatement() {
    final StringBuilder sb =
        new StringBuilder().append("INSERT INTO ").append(mName.toUpperCase()).append("\n(");

    String sep = "";
    for (final Column col : mColumns) {
      sb.append(sep).append("\"").append(col.name.toUpperCase()).append("\"");
      sep = ", ";
    }
    sb.append(")\nVALUES(")
        .append(String.join(", ", Collections.nCopies(mColumns.size(), "?")))
        .append(")");
    return sb.toString();
  }

  public static class Blob implements SQLType {
    public String toString() {
      return "BLOB";
    }

    public int getType() {
      return Types.BLOB;
    }
  }

  public static class Boolean implements SQLType {
    public String toString() {
      return "CHAR(1)";
    }

    public int getType() {
      return Types.CHAR;
    }
  }

  public static class Clob implements SQLType {
    public String toString() {
      return "CLOB";
    }

    public int getType() {
      return Types.CLOB;
    }
  }

  public static class Integer implements SQLType {
    public String toString() {
      return "NUMBER(20)";
    }

    public int getType() {
      return Types.BIGINT;
    }
  }

  public static class Timestamp implements SQLType {
    public String toString() {
      return "TIMESTAMP WITH TIME ZONE";
    }

    public int getType() {
      return Types.TIMESTAMP_WITH_TIMEZONE;
    }
  }

  public static class Varchar2 implements SQLType {
    private int mLength;

    public Varchar2(final int maxLength) {
      mLength = maxLength;
    }

    public String toString() {
      return String.format("VARCHAR2(%d)", mLength);
    }

    public int getType() {
      return Types.VARCHAR;
    }
  }
}
