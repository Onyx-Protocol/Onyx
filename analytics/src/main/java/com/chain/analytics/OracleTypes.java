package com.chain.analytics;

import java.sql.Types;

public class OracleTypes {

  /**
   * parse takes a string representing a Chain Analytics type and returns
   * the corresponding Oracle sql type. If parsing fails or any reason,
   * this function returns null.
   *
   * @param  typeString the Chain Analytics type
   * @return            the corresponding Oracle type
   */
  public static Schema.SQLType parse(final String typeString) {
    String[] tokens = typeString.trim().toLowerCase().split("[()]{1}");
    if (tokens.length == 0) {
      return null;
    }

    switch (tokens[0]) {
      case "bigint":
        if (tokens.length > 1) {
          return null;
        }
        return new BigInteger();
      case "blob":
        if (tokens.length > 1) {
          return null;
        }
        return new Blob();
      case "boolean":
        if (tokens.length > 1) {
          return null;
        }
        return new Boolean();
      case "clob":
        if (tokens.length > 1) {
          return null;
        }
        return new Clob();
      case "timestamp":
        if (tokens.length > 1) {
          return null;
        }
        return new Timestamp();
      case "varchar":
        if (tokens.length < 2) {
          return null;
        }
        try {
          final int width = Integer.parseInt(tokens[1]);
          if (width > 4000) {
            return null;
          }
          return new Varchar2(width);
        } catch (NumberFormatException ex) {
          return null;
        }
    }
    return null;
  }

  public static class BigInteger implements Schema.SQLType {
    public String toString() {
      return "bigint";
    }

    public String toDDL() {
      return "NUMBER(20)";
    }

    public int getType() {
      return Types.BIGINT;
    }
  }

  public static class Blob implements Schema.SQLType {
    public String toString() {
      return "blob";
    }

    public String toDDL() {
      return "BLOB";
    }

    public int getType() {
      return Types.BLOB;
    }
  }

  public static class Boolean implements Schema.SQLType {
    public String toString() {
      return "boolean";
    }

    public String toDDL() {
      return "CHAR(1)";
    }

    public int getType() {
      return Types.CHAR;
    }
  }

  public static class Clob implements Schema.SQLType {
    public String toString() {
      return "clob";
    }

    public String toDDL() {
      return "CLOB";
    }

    public int getType() {
      return Types.CLOB;
    }
  }

  public static class Timestamp implements Schema.SQLType {
    public String toString() {
      return "timestamp";
    }

    public String toDDL() {
      return "TIMESTAMP WITH TIME ZONE";
    }

    public int getType() {
      return Types.TIMESTAMP_WITH_TIMEZONE;
    }
  }

  public static class Varchar2 implements Schema.SQLType {
    private final int mLength;

    public Varchar2(final int maxLength) {
      mLength = maxLength;
    }

    public String toString() {
      return String.format("varchar(%d)", mLength);
    }

    public String toDDL() {
      return String.format("VARCHAR2(%d)", mLength);
    }

    public int getType() {
      return Types.VARCHAR;
    }
  }
}
