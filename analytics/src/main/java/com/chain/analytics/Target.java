package com.chain.analytics;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;

import javax.sql.DataSource;
import java.sql.*;
import java.util.*;

/**
 * Target represents a target destination for the Chain Analytics importer.
 */
public class Target {
  private static final String SQL_SELECT_CUSTOM_COLUMNS =
      "SELECT \"TABLE\", \"NAME\", \"TYPE\", \"PATH\" FROM custom_columns";
  private static final String SQL_DELETE_CUSTOM_COLUMN =
      "DELETE FROM custom_columns WHERE \"TABLE\" = ? AND \"NAME\" = ?";
  private static final String SQL_INSERT_CUSTOM_COLUMN =
      "INSERT INTO custom_columns (\"TABLE\", \"NAME\", \"TYPE\", \"PATH\") VALUES(?, ?, ?, ?)";
  private static final String SQL_CUSTOM_COLUMN_DDL =
      "CREATE TABLE custom_columns (\n"
          + "\"TABLE\" VARCHAR2(64),\n"
          + "\"NAME\" VARCHAR2(64),\n"
          + "\"TYPE\" VARCHAR2(64),\n"
          + "\"PATH\" VARCHAR2(4000))";
  private static final Logger logger = LogManager.getLogger();

  private Config mConfig;
  private Schema mTransactionsTbl;
  private Schema mTransactionInputsTbl;
  private Schema mTransactionOutputsTbl;
  private final DataSource mDataSource;

  public Target(final DataSource ds) throws Config.InvalidConfigException, SQLException {
    mDataSource = ds;
    mConfig = loadConfig(ds);
    reconstructSchemas();
    initializeSchema();
  }

  public Config getConfig() {
    return mConfig;
  }

  public DataSource getDataSource() {
    return mDataSource;
  }

  public Schema getTransactionsSchema() {
    return mTransactionsTbl;
  }

  public Schema getInputsSchema() {
    return mTransactionInputsTbl;
  }

  public Schema getOutputsSchema() {
    return mTransactionOutputsTbl;
  }

  /**
   * Migrate takes a new Chain Analytics configuration and migrates
   * the database to adopt the new config.
   * @param newConfig a Chain Analytics configuration
   */
  public void migrate(Config newConfig) throws SQLException {
    // Figure out what columns we need to add and remove to get
    // to newConfig's state.
    final Config.Migration migration = mConfig.diff(newConfig);

    try (Connection conn = mDataSource.getConnection()) {
      // Remove columns that are not in the new config.
      for (Map.Entry<String, List<Config.CustomColumn>> e : migration.removed.entrySet()) {
        if (e.getValue().isEmpty()) {
          continue;
        }

        final StringBuilder sb = new StringBuilder();
        sb.append("ALTER TABLE ");
        sb.append(e.getKey());
        sb.append(" DROP (");
        String sep = "";
        for (Config.CustomColumn drop : e.getValue()) {
          sb.append(sep);
          sb.append(drop.name.toUpperCase());
          sep = ", ";
        }
        sb.append(")");

        logger.info("Running migration: {}", sb.toString());

        // remove them from the configuration
        removeCustomColumns(conn, e.getKey(), e.getValue());
        // and drop them from the schema
        try (PreparedStatement ps = conn.prepareStatement(sb.toString())) {
          ps.executeUpdate();
        }
      }

      // Add columns that are in the new config but not in the current.
      for (Map.Entry<String, List<Config.CustomColumn>> e : migration.added.entrySet()) {
        if (e.getValue().isEmpty()) {
          continue;
        }

        final StringBuilder sb = new StringBuilder();
        sb.append("ALTER TABLE ");
        sb.append(e.getKey());
        sb.append(" ADD (");
        String sep = "";
        for (Config.CustomColumn add : e.getValue()) {
          sb.append(sep);
          sb.append("\"");
          sb.append(add.name.toUpperCase());
          sb.append("\" ");
          sb.append(add.type.toDDL());
          sep = ", ";
        }
        sb.append(")");

        logger.info("Running migration: {}", sb.toString());

        // add them to the configuration table
        insertCustomColumns(conn, e.getKey(), e.getValue());
        // and add them to the schema
        try (PreparedStatement ps = conn.prepareStatement(sb.toString())) {
          ps.executeUpdate();
        }
      }
    }

    mConfig = newConfig;
    reconstructSchemas();
  }

  private void initializeSchema() throws SQLException {
    try (Connection conn = mDataSource.getConnection()) {
      createTableIfNotExists(conn, SQL_CUSTOM_COLUMN_DDL);
      createTableIfNotExists(conn, mTransactionsTbl.getDDLStatement());
      createTableIfNotExists(conn, mTransactionInputsTbl.getDDLStatement());
      createTableIfNotExists(conn, mTransactionOutputsTbl.getDDLStatement());
    }
  }

  private boolean createTableIfNotExists(final Connection conn, final String query)
      throws SQLException {
    logger.info("Creating table: \n{}", query);
    try (PreparedStatement ps = conn.prepareStatement(query)) {
      ps.executeUpdate();
    } catch (SQLSyntaxErrorException ex) {
      // If "ORA-00955: name is already used by an existing object",
      // the table already exists. Otherwise, it's an unexpected exception.
      if (ex.getErrorCode() != 955) {
        throw ex;
      }
      return false;
    }
    return true;
  }

  /**
   * loadConfig retrieves the Chain Analytics configuration from the Oracle database.
   * If there is no current configuration, load returns an empty Config.
   *
   * @return    the current Chain Analytics configuration
   */
  private static Config loadConfig(final DataSource ds)
      throws Config.InvalidConfigException, SQLException {
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
          throw new Config.InvalidConfigException(String.format("Unknown column type %s", rawType));
        }
        final JsonPath path = new JsonPath(rawPath);

        switch (tbl.toLowerCase()) {
          case "transactions":
            config.transactionColumns.add(new Config.CustomColumn(name, type, path));
            break;
          case "transaction_inputs":
            config.inputColumns.add(new Config.CustomColumn(name, type, path));
            break;
          case "transaction_outputs":
            config.outputColumns.add(new Config.CustomColumn(name, type, path));
            break;
          default:
            throw new Config.InvalidConfigException(String.format("Unknown table %s", tbl));
        }
      }
    } catch (SQLException ex) {
      if (ex.getErrorCode() == 942) {
        // ORA-00942 table or view does not exist.
        return config;
      }
      throw ex;
    }
    return config;
  }

  private void reconstructSchemas() throws SQLException {
    Schema.Builder transactionsBuilder =
        new Schema.Builder("transactions")
            .setPrimaryKey(Collections.singletonList("id"))
            .addColumn("id", new OracleTypes.Varchar2(64))
            .addColumn("block_height", new OracleTypes.BigInteger())
            .addColumn("timestamp", new OracleTypes.Timestamp())
            .addColumn("position", new OracleTypes.BigInteger())
            .addColumn("local", new OracleTypes.Boolean())
            .addColumn("reference_data", new OracleTypes.Blob())
            .addColumn("data", new OracleTypes.Blob());
    for (Config.CustomColumn col : mConfig.transactionColumns) {
      transactionsBuilder.addColumn(col.name, col.type);
    }

    Schema.Builder inputsBuilder =
        new Schema.Builder("transaction_inputs")
            .setPrimaryKey(Arrays.asList("transaction_id", "index"))
            .addColumn("transaction_id", new OracleTypes.Varchar2(64))
            .addColumn("index", new OracleTypes.BigInteger())
            .addColumn("type", new OracleTypes.Varchar2(64))
            .addColumn("asset_id", new OracleTypes.Varchar2(64))
            .addColumn("asset_alias", new OracleTypes.Varchar2(2000))
            .addColumn("asset_definition", new OracleTypes.Blob())
            .addColumn("asset_tags", new OracleTypes.Blob())
            .addColumn("local_asset", new OracleTypes.Boolean())
            .addColumn("amount", new OracleTypes.BigInteger())
            .addColumn("account_id", new OracleTypes.Varchar2(64))
            .addColumn("account_alias", new OracleTypes.Varchar2(2000))
            .addColumn("account_tags", new OracleTypes.Blob())
            .addColumn("issuance_program", new OracleTypes.Clob())
            .addColumn("reference_data", new OracleTypes.Blob())
            .addColumn("local", new OracleTypes.Boolean())
            .addColumn("spent_output_id", new OracleTypes.Varchar2(64));
    for (Config.CustomColumn col : mConfig.inputColumns) {
      inputsBuilder.addColumn(col.name, col.type);
    }

    Schema.Builder outputsBuilder =
        new Schema.Builder("transaction_outputs")
            .setPrimaryKey(Collections.singletonList("output_id"))
            .addUniqueConstraint(Arrays.asList("transaction_id", "index"))
            .addColumn("transaction_id", new OracleTypes.Varchar2(64))
            .addColumn("index", new OracleTypes.BigInteger())
            .addColumn("output_id", new OracleTypes.Varchar2(64))
            .addColumn("type", new OracleTypes.Varchar2(64))
            .addColumn("purpose", new OracleTypes.Varchar2(64))
            .addColumn("asset_id", new OracleTypes.Varchar2(64))
            .addColumn("asset_alias", new OracleTypes.Varchar2(2000))
            .addColumn("asset_definition", new OracleTypes.Blob())
            .addColumn("asset_tags", new OracleTypes.Blob())
            .addColumn("local_asset", new OracleTypes.Boolean())
            .addColumn("amount", new OracleTypes.BigInteger())
            .addColumn("account_id", new OracleTypes.Varchar2(64))
            .addColumn("account_alias", new OracleTypes.Varchar2(2000))
            .addColumn("account_tags", new OracleTypes.Blob())
            .addColumn("control_program", new OracleTypes.Clob())
            .addColumn("reference_data", new OracleTypes.Blob())
            .addColumn("local", new OracleTypes.Boolean())
            .addColumn("spent", new OracleTypes.Boolean());
    for (Config.CustomColumn col : mConfig.outputColumns) {
      inputsBuilder.addColumn(col.name, col.type);
    }

    mTransactionsTbl = transactionsBuilder.build();
    mTransactionInputsTbl = inputsBuilder.build();
    mTransactionOutputsTbl = outputsBuilder.build();
  }

  private static void removeCustomColumns(
      final Connection conn, final String table, final List<Config.CustomColumn> remove)
      throws SQLException {
    try (PreparedStatement ps = conn.prepareStatement(SQL_DELETE_CUSTOM_COLUMN)) {
      for (final Config.CustomColumn cc : remove) {
        ps.setString(1, table);
        ps.setString(2, cc.name);
        ps.executeUpdate();
      }
    }
  }

  private static void insertCustomColumns(
      final Connection conn, final String table, final List<Config.CustomColumn> insert)
      throws SQLException {
    try (PreparedStatement ps = conn.prepareStatement(SQL_INSERT_CUSTOM_COLUMN)) {
      for (final Config.CustomColumn cc : insert) {
        ps.setString(1, table);
        ps.setString(2, cc.name);
        ps.setString(3, cc.type.toString());
        ps.setString(4, cc.jsonPath.toString());
        ps.executeUpdate();
      }
    }
  }
}
