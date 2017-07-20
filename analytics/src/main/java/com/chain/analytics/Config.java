package com.chain.analytics;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.util.ArrayList;
import java.util.List;

public class Config {
  private static final String SQL_SELECT_CUSTOM_COLUMNS =
      "SELECT \"TABLE\", \"NAME\", \"TYPE\", \"PATH\" FROM custom_columns";

  final List<CustomColumn> transactionColumns;
  final List<CustomColumn> inputColumns;
  final List<CustomColumn> outputColumns;

  private Config() {
    transactionColumns = new ArrayList<>();
    inputColumns = new ArrayList<>();
    outputColumns = new ArrayList<>();
  }

  /**
   * load pulls in the Chain Analytics configuration from the Oracle database.
   * If there is no current configuration, load returns null.
   *
   * @param  ds the Oracle database datasource
   * @return    the current Chain Analytics configuration
   */
  public static Config load(final DataSource ds) throws InvalidConfigException, SQLException {
    final Config config = new Config();
    try (Connection conn = ds.getConnection();
        Statement stmt = conn.createStatement();
        ResultSet rs = stmt.executeQuery(SQL_SELECT_CUSTOM_COLUMNS)) {
      while (rs.next()) {
        final String tbl = rs.getString("TABLE");
        final String name = rs.getString("NAME");
        final String rawType = rs.getString("TYPE");
        final String rawPath = rs.getString("PATH");

        final Schema.SQLType type = OracleTypes.parse(rawType);
        if (type == null) {
          throw new InvalidConfigException(String.format("Unknown column type %s", rawType));
        }
        final JsonPath path = new JsonPath(rawPath);

        switch (tbl.toLowerCase()) {
          case "transactions":
            config.transactionColumns.add(new CustomColumn(name, type, path));
            break;
          case "transaction_inputs":
            config.inputColumns.add(new CustomColumn(name, type, path));
            break;
          case "transaction_outputs":
            config.outputColumns.add(new CustomColumn(name, type, path));
            break;
          default:
            throw new InvalidConfigException(String.format("Unknown table %s", tbl));
        }
      }
    } catch (SQLException ex) {
      if (ex.getErrorCode() == 942) {
        // ORA-00942 table or view does not exist.
        return null;
      }
      throw ex;
    }
    return config;
  }

  public static class CustomColumn {
    final String name;
    final Schema.SQLType type;
    final JsonPath jsonPath;

    public CustomColumn(String name, Schema.SQLType type, JsonPath jsonPath) {
      this.name = name;
      this.type = type;
      this.jsonPath = jsonPath;
    }
  }

  public static class InvalidConfigException extends Exception {
    private static final long serialVersionUID = 201775336232807009L;

    public InvalidConfigException(String message) {
      super(message);
    }
  }
}
