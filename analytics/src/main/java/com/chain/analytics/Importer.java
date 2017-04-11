package analytics;

import com.chain.http.Client;
import com.chain.api.PagedItems;
import com.chain.api.Query;
import com.chain.api.Transaction;
import com.chain.api.Transaction.QueryBuilder;
import com.chain.exception.APIException;
import com.chain.exception.ChainException;
import com.google.gson.Gson;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.SQLException;
import java.sql.SQLSyntaxErrorException;
import java.sql.Timestamp;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;
import javax.sql.DataSource;

import org.apache.logging.log4j.Logger;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.ThreadContext;

/**
 * Importer is responsible for reading transactions from a Chain Core
 * transaction feed and writing them to an Oracle database.
 */
public class Importer {

  private static final String TRUE = "1";
  private static final String FALSE = "0";
  private static final long defaultTimeoutMillis = 30 * 1000; // 30 seconds

  private static final Logger logger = LogManager.getLogger();
  private static final Gson gson = new Gson();

  private Client mChain;
  private DataSource mDataSource;
  private Transaction.Feed mFeed;
  private Schema mTransactionsTbl;
  private Schema mTransactionInputsTbl;
  private Schema mTransactionOutputsTbl;

  /**
   * connect initializes an Importer using the transaction feed specified
   * by the alias. If the feed doesn't yet exist, it will be created. It
   * does not begin syncing yet.
   * @param client    a client for the Chain Core
   * @param ds        the database to populate
   * @param feedAlias the alias of the transaction feed to use
   * @return          the initialized transaction importer
   */
  public static Importer connect(final Client client, final DataSource ds, final String feedAlias)
      throws ChainException, SQLException {
    // Create or load the transaction feed for the provided alias.
    try {
      Transaction.Feed.create(client, feedAlias, "");
    } catch (APIException ex) {
      // CH050 means the transaction feed already existed. If that's
      // the case, ignore the exception because we'll retrieve the
      // feed down below by its alias.
      if (!"CH050".equals(ex.code)) {
        logger.catching(ex);
        throw ex;
      }
      logger.info("Transaction feed {} already exists", feedAlias);
    }
    final Transaction.Feed feed = Transaction.Feed.getByAlias(client, feedAlias);
    logger.info("Using transaction feed {} starting at cursor {}", feed.id, feed.after);

    // Initialize the schema based on the configuration.
    final Importer importer = new Importer(client, ds, feed);
    importer.initializeSchema();
    return importer;
  }

  private Importer(final Client client, final DataSource ds, final Transaction.Feed feed) {
    mChain = client;
    mDataSource = ds;
    mFeed = feed;
  }

  void initializeSchema() throws SQLException {

    mTransactionsTbl =
        new Schema.Builder("transactions")
            .setPrimaryKey(Arrays.asList("id"))
            .addColumn("id", new Schema.Varchar2(64))
            .addColumn("block_height", new Schema.Integer())
            .addColumn("timestamp", new Schema.Timestamp())
            .addColumn("position", new Schema.Integer())
            .addColumn("local", new Schema.Boolean())
            .addColumn("reference_data", new Schema.Blob())
            .addColumn("data", new Schema.Blob())
            .build();

    mTransactionInputsTbl =
        new Schema.Builder("transaction_inputs")
            .setPrimaryKey(Arrays.asList("transaction_id", "index"))
            .addColumn("transaction_id", new Schema.Varchar2(64))
            .addColumn("index", new Schema.Integer())
            .addColumn("type", new Schema.Varchar2(64))
            .addColumn("asset_id", new Schema.Varchar2(64))
            .addColumn("asset_alias", new Schema.Varchar2(2000))
            .addColumn("asset_definition", new Schema.Blob())
            .addColumn("asset_tags", new Schema.Blob())
            .addColumn("local_asset", new Schema.Boolean())
            .addColumn("amount", new Schema.Integer())
            .addColumn("account_id", new Schema.Varchar2(64))
            .addColumn("account_alias", new Schema.Varchar2(2000))
            .addColumn("account_tags", new Schema.Blob())
            .addColumn("issuance_program", new Schema.Clob())
            .addColumn("reference_data", new Schema.Blob())
            .addColumn("local", new Schema.Boolean())
            .addColumn("spent_output_id", new Schema.Varchar2(64))
            .build();

    mTransactionOutputsTbl =
        new Schema.Builder("transaction_outputs")
            .setPrimaryKey(Arrays.asList("output_id"))
            .addColumn("transaction_id", new Schema.Varchar2(64))
            .addColumn("index", new Schema.Integer())
            .addColumn("output_id", new Schema.Varchar2(64))
            .addColumn("type", new Schema.Varchar2(64))
            .addColumn("purpose", new Schema.Varchar2(64))
            .addColumn("asset_id", new Schema.Varchar2(64))
            .addColumn("asset_alias", new Schema.Varchar2(2000))
            .addColumn("asset_definition", new Schema.Blob())
            .addColumn("asset_tags", new Schema.Blob())
            .addColumn("local_asset", new Schema.Boolean())
            .addColumn("amount", new Schema.Integer())
            .addColumn("account_id", new Schema.Varchar2(64))
            .addColumn("account_alias", new Schema.Varchar2(2000))
            .addColumn("account_tags", new Schema.Blob())
            .addColumn("control_program", new Schema.Clob())
            .addColumn("reference_data", new Schema.Blob())
            .addColumn("local", new Schema.Boolean())
            .addColumn("spent", new Schema.Boolean())
            .build();

    try (Connection conn = mDataSource.getConnection()) {
      createTableIfNotExists(conn, mTransactionsTbl.getDDLStatement());
      createTableIfNotExists(conn, mTransactionInputsTbl.getDDLStatement());
      createTableIfNotExists(conn, mTransactionOutputsTbl.getDDLStatement());
    }

    // TODO(jackson): Perform some kind of checksuming on the DDL
    // statements so that we notice if the existing tables were created
    // from a different configuration? Or store the configuration
    // itself in Oracle so that they *must* run a program to reconfigure.
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
   * Processes transactions, reading them from the transaction feed
   * and inserting them into the configured Oracle database. This
   * function blocks indefinitely. Errors are logged to log4j2.
   *
   * TODO(jackson): Support propagating persistent errors?
   */
  public void process() {
    ThreadContext.put("feed_id", mFeed.id);
    ThreadContext.put("feed_filter", mFeed.filter);

    for (; ; ) {
      try {
        ThreadContext.put("feed_after", mFeed.after);

        // Retrieve another page of transactions matching the feed.
        final Transaction.Items page =
            new QueryBuilder()
                .setFilter(mFeed.filter)
                .setAfter(mFeed.after)
                .setTimeout(defaultTimeoutMillis)
                .setAscendingWithLongPoll()
                .execute(mChain);

        // Commit the batch of transactions to the database.
        try {
          processBatch(page.list);
        } catch (SQLException ex) {
          logger.catching(ex);
          // Skip the ack so we re-fetch this page and try again.
          // TODO(jackson): Do we need any backoff? Don't want to
          // hammer Chain Core or the Oracle database if there's
          // an issue.
          // TODO(jackson): Add retry logic in processBatch so that
          // we don't re-query Chain Core if we don't need to?
          continue;
        }

        // Acknowledge that we've processed the entire page.
        Map<String, Object> requestBody = new TreeMap<>();
        requestBody.put("id", mFeed.id);
        requestBody.put("previous_after", mFeed.after);
        requestBody.put("after", page.next.after);
        mFeed = mChain.request("update-transaction-feed", requestBody, Transaction.Feed.class);
      } catch (APIException ex) {
        // If there was an issue retrieving the transactions,
        // log it. If the request just timed out, no matching
        // transactions were committed so just silently ignore it.
        if (!"CH001".equals(ex.code)) {
          logger.catching(ex);
        }
      } catch (ChainException ex) {
        logger.catching(ex);
      } finally {
        ThreadContext.remove("feed_after");
      }
    }
  }

  // processBatch inserts a batch of transactions into the Oracle
  // database.
  //
  // TODO(jackson): Ideally we'd call executeBatch once per prepared-statement
  // per call to processBatch, but handling the unique constraint violations
  // gets trickier. We might be able to write a PL/SQL function that ignores
  // the ORA-00001 exception in Oracle instead of client-side to get around
  // that.
  void processBatch(final List<Transaction> transactions) throws SQLException {
    final String insertTxQ = mTransactionsTbl.getInsertStatement();
    final String insertInputQ = mTransactionInputsTbl.getInsertStatement();
    final String insertOutputQ = mTransactionOutputsTbl.getInsertStatement();

    try (Connection conn = mDataSource.getConnection();
        PreparedStatement psTx = conn.prepareStatement(insertTxQ);
        PreparedStatement psIn = conn.prepareStatement(insertInputQ);
        PreparedStatement psOut = conn.prepareStatement(insertOutputQ)) {

      // Manage our own SQL transactions so we can make this whole
      // blockchain transaction atomic.
      conn.setAutoCommit(false);

      for (Transaction tx : transactions) {
        try {
          ThreadContext.put("tx_id", tx.id);
          logger.debug("Importing transaction {}", tx.id);

          // Insert the transaction itself.
          psTx.setString(1, tx.id);
          psTx.setLong(2, tx.blockHeight);
          psTx.setTimestamp(3, new Timestamp(tx.timestamp.getTime()));
          psTx.setInt(4, tx.position);
          psTx.setString(5, "yes".equals(tx.isLocal) ? TRUE : FALSE);
          psTx.setBlob(6, asJsonBlob(tx.referenceData));
          psTx.setBlob(7, asJsonBlob(tx));

          // TODO(jackson): We can't use addBatch in Oracle
          // earlier than 12.1. If customers need support for
          // Oracle < 12.1, we'll need to use the deprecated
          // OraclePreparedStatement.setExecuteBatch method:
          // http://docs.oracle.com/cd/B28359_01/java.111/b31224/oraperf.htm
          psTx.addBatch();

          // Insert each of the inputs.
          for (int i = 0; i < tx.inputs.size(); i++) {
            final Transaction.Input input = tx.inputs.get(i);

            psIn.setString(1, tx.id);
            psIn.setInt(2, i);
            psIn.setString(3, input.type);
            psIn.setString(4, input.assetId);
            psIn.setString(5, input.assetAlias);
            psIn.setBlob(6, asJsonBlob(input.assetDefinition));
            psIn.setBlob(7, asJsonBlob(input.assetTags));
            psIn.setString(8, "yes".equals(input.assetIsLocal) ? TRUE : FALSE);
            psIn.setLong(9, input.amount);
            psIn.setString(10, input.accountId);
            psIn.setString(11, input.accountAlias);
            psIn.setBlob(12, asJsonBlob(input.accountTags));
            psIn.setString(13, input.issuanceProgram); // clob
            psIn.setBlob(14, asJsonBlob(input.referenceData));
            psIn.setString(15, "yes".equals(input.isLocal) ? TRUE : FALSE);
            psIn.setString(16, input.spentOutputId);

            psIn.addBatch();
          }

          // Insert each of the outputs.
          for (int i = 0; i < tx.outputs.size(); i++) {
            final Transaction.Output output = tx.outputs.get(i);

            psOut.setString(1, tx.id);
            psOut.setInt(2, i);
            psOut.setString(3, output.id);
            psOut.setString(4, output.type);
            psOut.setString(5, output.purpose);
            psOut.setString(6, output.assetId);
            psOut.setString(7, output.assetAlias);
            psOut.setBlob(8, asJsonBlob(output.assetDefinition));
            psOut.setBlob(9, asJsonBlob(output.assetTags));
            psOut.setString(10, "yes".equals(output.assetIsLocal) ? TRUE : FALSE);
            psOut.setLong(11, output.amount);
            psOut.setString(12, output.accountId);
            psOut.setString(13, output.accountAlias);
            psOut.setBlob(14, asJsonBlob(output.accountTags));
            psOut.setString(15, output.controlProgram); // clob
            psOut.setBlob(16, asJsonBlob(output.referenceData));
            psOut.setString(17, "yes".equals(output.isLocal) ? TRUE : FALSE);
            psOut.setString(18, FALSE);

            psOut.addBatch();
          }

          // Commit the entire blockchain transaction at once.
          psTx.executeBatch();
          psIn.executeBatch();
          psOut.executeBatch();
          conn.commit();
        } catch (SQLException ex) {
          // We can hit a unique constraint violation (ORA-00001)
          // iff we already processed these transactions but have not
          // yet successfully acked them. If that's the case, just
          // fallthrough to the acknowledgement below.
          if (ex.getErrorCode() == 1) {
            logger.info("Processed transaction twice; ignoring");
          } else {
            throw ex;
          }
        } finally {
          ThreadContext.remove("tx_id");
        }
      }
    }
  }

  private static InputStream asJsonBlob(final Object obj) {
    if (obj == null) return null;
    return new ByteArrayInputStream(gson.toJson(obj).getBytes(StandardCharsets.UTF_8));
  }
}
